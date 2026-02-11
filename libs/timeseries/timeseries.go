package timeseries

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"gorm.io/gorm"
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
	Name         string
	Retention    SamplingPolicy   // main ingestion retention policy
	DownSampling []SamplingPolicy // additional downsampling policies
}

// dbTimeSeries represents one time series configuration
type dbTimeSeries struct {
	ID       uint               `gorm:"primaryKey;autoIncrement"`
	Name     string             `gorm:"uniqueIndex;not null"` // logical name, must be unique
	Policies []dbSamplingPolicy `gorm:"foreignKey:TimeSeriesID;constraint:OnDelete:CASCADE"`
}

func (dbTs *dbTimeSeries) mainPolicyID() uint {
	for _, p := range dbTs.Policies {
		if p.Name == "main" {
			return p.ID
		}
	}
	return 0
}

// dbSamplingPolicy defines a rollup/aggregation policy for a series
type dbSamplingPolicy struct {
	ID            uint          `gorm:"primaryKey"`
	TimeSeriesID  uint          `gorm:"not null;index:idx_series_policy,unique"` // FK to dbTimeSeries.id
	Name          string        `gorm:"not null;index:idx_series_policy,unique"` // unique per series
	Precision     time.Duration `gorm:"not null"`
	Retention     time.Duration `gorm:"not null"`
	AggregationFn string        `gorm:"not null"`
}

// samplingPolicyName used internally to generate a unique identifying name for sampling policies
func samplingPolicyName(retention, precision time.Duration, aggrFn string) string {
	return fmt.Sprintf("%s_%s_%s", retention.String(), precision.String(), aggrFn)
}

const mainPolicyName = "main"

// RegisterSeries inserts or updates a series.
// If it exists, all DownSamplingPolicies are replaced.
func (ts *Registry) RegisterSeries(series TimeSeries) error {
	return ts.db.Transaction(func(tx *gorm.DB) error {
		existing, err := findOrCreateSeries(tx, series.Name)
		if err != nil {
			return err
		}

		// Load existing policies
		var existingPolicies []dbSamplingPolicy
		if err := tx.Where("time_series_id = ?", existing.ID).Find(&existingPolicies).Error; err != nil {
			return fmt.Errorf("load existing policies: %w", err)
		}

		existingMap := make(map[string]dbSamplingPolicy)
		for _, p := range existingPolicies {
			existingMap[p.Name] = p
		}
		seen := make(map[string]bool)

		// Handle main retention policy
		seen[mainPolicyName] = true
		mainPolicyData := series.Retention
		if existingMain, ok := existingMap[mainPolicyName]; ok {
			// Update existing main policy if changed
			if existingMain.Precision != mainPolicyData.Precision ||
				existingMain.Retention != mainPolicyData.Retention ||
				existingMain.AggregationFn != mainPolicyData.AggregationFn {
				existingMain.Precision = mainPolicyData.Precision
				existingMain.Retention = mainPolicyData.Retention
				existingMain.AggregationFn = mainPolicyData.AggregationFn
				if err := tx.Save(&existingMain).Error; err != nil {
					return fmt.Errorf("update main policy: %w", err)
				}
			}
		} else {
			// Create new main policy
			mainPolicy := dbSamplingPolicy{
				Name:          mainPolicyName,
				TimeSeriesID:  existing.ID,
				Precision:     mainPolicyData.Precision,
				Retention:     mainPolicyData.Retention,
				AggregationFn: mainPolicyData.AggregationFn,
			}
			if err := tx.Create(&mainPolicy).Error; err != nil {
				return fmt.Errorf("create main policy: %w", err)
			}
		}

		// Handle downsampling policies, main policy is not habdled here
		if err := upsertPolicies(tx, existing.ID, existingMap, seen, series.DownSampling); err != nil {
			return err
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

func findOrCreateSeries(tx *gorm.DB, name string) (dbTimeSeries, error) {
	var existing dbTimeSeries
	err := tx.Where("name = ?", name).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		newSeries := dbTimeSeries{Name: name}
		if err := tx.Create(&newSeries).Error; err != nil {
			return dbTimeSeries{}, fmt.Errorf("create series: %w", err)
		}
		return newSeries, nil
	case err != nil:
		return dbTimeSeries{}, fmt.Errorf("find series: %w", err)
	default:
		return existing, nil
	}
}

func upsertPolicies(tx *gorm.DB, seriesID uint, existingMap map[string]dbSamplingPolicy, seen map[string]bool, sampling []SamplingPolicy) error {
	for _, p := range sampling {
		name := samplingPolicyName(p.Retention, p.Precision, p.AggregationFn)
		seen[name] = true

		if existing, ok := existingMap[name]; ok {
			if existing.Precision == p.Precision && existing.Retention == p.Retention && existing.AggregationFn == p.AggregationFn {
				continue // no change
			}

			existing.Precision = p.Precision
			existing.Retention = p.Retention
			existing.AggregationFn = p.AggregationFn
			if err := tx.Save(&existing).Error; err != nil {
				return fmt.Errorf("update policy %s: %w", name, err)
			}
			continue
		}

		newPolicy := dbSamplingPolicy{
			Name:          name,
			TimeSeriesID:  seriesID,
			Precision:     p.Precision,
			Retention:     p.Retention,
			AggregationFn: p.AggregationFn,
		}
		if err := tx.Create(&newPolicy).Error; err != nil {
			return fmt.Errorf("create policy %s: %w", name, err)
		}
	}
	return nil
}

// ListSeries returns all series with their downsampling policies
func (ts *Registry) ListSeries() ([]TimeSeries, error) {
	var dbSeries []dbTimeSeries
	if err := ts.db.Preload("Policies").Find(&dbSeries).Error; err != nil {
		return nil, err
	}

	result := make([]TimeSeries, len(dbSeries))
	for i, s := range dbSeries {
		out := TimeSeries{
			Name: s.Name,
		}

		// Separate main retention policy from downsampling policies
		var downsampling []SamplingPolicy
		for _, p := range s.Policies {
			policy := SamplingPolicy{
				Precision:     p.Precision,
				Retention:     p.Retention,
				AggregationFn: p.AggregationFn,
			}
			if p.Name == mainPolicyName {
				out.Retention = policy
			} else {
				downsampling = append(downsampling, policy)
			}
		}
		out.DownSampling = downsampling
		result[i] = out
	}

	return result, nil
}

// GetSeries loads a series with its DownSampling policies preloaded
func (ts *Registry) GetSeries(name string) (TimeSeries, error) {
	var series dbTimeSeries
	err := ts.db.Preload("Policies").Where("name = ?", name).First(&series).Error
	if err != nil {
		return TimeSeries{}, err
	}
	ret := TimeSeries{
		Name: series.Name,
	}

	// Separate main retention policy from downsampling policies
	var downsampling []SamplingPolicy
	for _, p := range series.Policies {
		policy := SamplingPolicy{
			Precision:     p.Precision,
			Retention:     p.Retention,
			AggregationFn: p.AggregationFn,
		}
		if p.Name == mainPolicyName {
			ret.Retention = policy
		} else {
			downsampling = append(downsampling, policy)
		}
	}
	ret.DownSampling = downsampling
	return ret, err
}

// Cleanup removes old records beyond the retention period
// todo, should do cleanup and downsampling as background job
func (ts *Registry) Cleanup(ctx context.Context) error {

	series, err := ts.ListSeries()
	if err != nil {
		return fmt.Errorf("unable to clean time series: %w", err)
	}

	// sort series sampling by retention duration

	// calculate new values by sampling down if not exits already

	for _, item := range series {
		err = ts.cleanOneSeries(ctx, item)
	}

	// sample down new data

	spew.Dump(series)

	//cutoff := time.Now().addTask(-ts.)
	//return ts.getDb.WithContext(ctx).
	//	Where("timestamp < ?", cutoff).
	//	Delete(&Record{}).Error

	return nil
}

// cleanOneSeries truncates all items in a single time series where the current time os newer than the retention
func (ts *Registry) cleanOneSeries(ctx context.Context, series TimeSeries) error {

	//policyName := samplingPolicyName(series.DownSampling)

	// cutof := time.Now().Add(series.DownSampling[0].Retention * -1)

	// delete all entries that are older than cut off on every sampling

	return nil
}
