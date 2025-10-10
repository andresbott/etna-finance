package timeseries

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type Registry struct {
	db *gorm.DB
}

// NewRegistry creates a new dbTimeSeries instance
func NewRegistry(db *gorm.DB) (*Registry, error) {
	if err := db.AutoMigrate(&dbTimeSeries{}, &dbSamplingPolicy{}, &dbRecord{}); err != nil {
		return nil, err
	}
	return &Registry{db: db}, nil
}

// SamplingPolicy defines a rollup/aggregation policy for a series
type SamplingPolicy struct {
	Retention     time.Duration
	Precision     time.Duration
	AggregationFn string
}

// TimeSeries represents one time series configuration
type TimeSeries struct {
	Name     string
	Sampling []SamplingPolicy
}

// dbTimeSeries represents one time series configuration
type dbTimeSeries struct {
	ID       uint               `gorm:"primaryKey;autoIncrement"`
	Name     string             `gorm:"uniqueIndex;not null"` // logical name, must be unique
	Sampling []dbSamplingPolicy `gorm:"foreignKey:TimeSeriesID;constraint:OnDelete:CASCADE"`
}

// dbSamplingPolicy defines a rollup/aggregation policy for a series
type dbSamplingPolicy struct {
	ID            uint          `gorm:"primaryKey"`
	TimeSeriesID  uint          `gorm:"not null;index:idx_series_policy,unique"` // FK to dbTimeSeries.ID
	Name          string        `gorm:"not null;index:idx_series_policy,unique"` // unique per series
	Precision     time.Duration `gorm:"not null"`
	Retention     time.Duration `gorm:"not null"`
	AggregationFn string        `gorm:"not null"`
}

// samplingPolicyName used internally to generate a unique identifying name for sampling policies
func samplingPolicyName(retention, precision time.Duration, aggrFn string) string {
	return fmt.Sprintf("%s_%s_%s,", retention.String(), precision.String(), aggrFn)
}

// RegisterSeries inserts or updates a series.
// If it exists, all DownSamplingPolicies are replaced.
func (ts *Registry) RegisterSeries(series TimeSeries) error {
	return ts.db.Transaction(func(tx *gorm.DB) error {
		var existing dbTimeSeries

		err := tx.Where("name = ?", series.Name).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new
			newSeries := dbTimeSeries{
				Name: series.Name,
			}
			if err := tx.Create(&newSeries).Error; err != nil {
				return fmt.Errorf("create series: %w", err)
			}
			existing = newSeries
		} else if err != nil {
			return fmt.Errorf("unable to find series series: %w", err)
		}

		// Load existing policies
		var existingPolicies []dbSamplingPolicy
		if err := tx.Where("time_series_id = ?", existing.ID).Find(&existingPolicies).Error; err != nil {
			return fmt.Errorf("load existing policies: %w", err)
		}

		existingMap := make(map[string]dbSamplingPolicy)
		for _, p := range existingPolicies {
			existingMap[samplingPolicyName(p.Retention, p.Precision, p.AggregationFn)] = p
		}

		// Process new/updated policies
		seen := make(map[string]bool)
		for _, p := range series.Sampling {
			samplingName := samplingPolicyName(p.Retention, p.Precision, p.AggregationFn)
			seen[samplingName] = true
			if existingPolicy, ok := existingMap[samplingName]; ok {
				// Exists → check if needs update
				if existingPolicy.Precision != p.Precision ||
					existingPolicy.Retention != p.Retention || existingPolicy.AggregationFn != p.AggregationFn {

					existingPolicy.Precision = p.Precision
					existingPolicy.Retention = p.Retention
					existingPolicy.AggregationFn = p.AggregationFn
					if err := tx.Save(&existingPolicy).Error; err != nil {
						return fmt.Errorf("unable to update policy %w", err)
					}
				}
			} else {
				// NewRegistry policy → insert
				newPolicy := dbSamplingPolicy{
					Name:          samplingName,
					TimeSeriesID:  existing.ID,
					Precision:     p.Precision,
					Retention:     p.Retention,
					AggregationFn: p.AggregationFn,
				}
				if err := tx.Create(&newPolicy).Error; err != nil {
					return fmt.Errorf("unable to create policy %w", err)
				}
			}
		}

		// Delete old policies not present anymore
		for _, old := range existingPolicies {
			if !seen[old.Name] {
				if err := tx.Delete(&old).Error; err != nil {
					return fmt.Errorf("delete policy %s: %w", old.Name, err)
				}
			}
		}
		return nil
	})
}

// ListSeries returns all series with their downsampling policies
func (ts *Registry) ListSeries() ([]TimeSeries, error) {
	var dbSeries []dbTimeSeries
	if err := ts.db.Preload("Sampling").Find(&dbSeries).Error; err != nil {
		return nil, err
	}

	result := make([]TimeSeries, len(dbSeries))
	for i, s := range dbSeries {
		out := TimeSeries{
			Name: s.Name,
		}

		down := make([]SamplingPolicy, len(s.Sampling))
		for j, d := range s.Sampling {
			down[j] = SamplingPolicy{
				Precision:     d.Precision,
				Retention:     d.Retention,
				AggregationFn: d.AggregationFn,
			}
		}

		out.Sampling = down
		result[i] = out
	}

	return result, nil
}

// GetSeries loads a series with its Sampling policies preloaded
func (ts *Registry) GetSeries(name string) (TimeSeries, error) {
	var series dbTimeSeries
	err := ts.db.Preload("Sampling").Where("name = ?", name).First(&series).Error

	ret := TimeSeries{
		Name: series.Name,
	}
	sampling := make([]SamplingPolicy, len(series.Sampling))
	for i, policy := range series.Sampling {
		sampling[i] = SamplingPolicy{
			Precision:     policy.Precision,
			Retention:     policy.Retention,
			AggregationFn: policy.AggregationFn,
		}
	}
	ret.Sampling = sampling
	return ret, err
}

// Cleanup removes old records beyond the retention period
// todo, should do cleanup and downsampling as baground job
func (ts *Registry) Cleanup(ctx context.Context) error {
	panic("implement me")
	//cutoff := time.Now().Add(-ts.)
	//return ts.db.WithContext(ctx).
	//	Where("timestamp < ?", cutoff).
	//	Delete(&Record{}).Error
}
