package csvimport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/andresbott/etna/internal/csvimport"
)

func (h *ImportHandler) PreviewCSV() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, fmt.Sprintf("unable to parse multipart form: %s", err.Error()), http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to get uploaded file: %s", err.Error()), http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to read uploaded file: %s", err.Error()), http.StatusBadRequest)
			return
		}

		skipRows := 0
		if v := r.FormValue("skipRows"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				skipRows = n
			}
		}

		profile := csvimport.ImportProfile{
			CsvSeparator:      r.FormValue("csvSeparator"),
			SkipRows:          skipRows,
			DateColumn:        r.FormValue("dateColumn"),
			DateFormat:        r.FormValue("dateFormat"),
			DescriptionColumn: r.FormValue("descriptionColumn"),
			AmountMode:        r.FormValue("amountMode"),
			AmountColumn:      r.FormValue("amountColumn"),
			CreditColumn:      r.FormValue("creditColumn"),
			DebitColumn:       r.FormValue("debitColumn"),
		}

		result, err := csvimport.ParsePreviewWithAutoDetect(data, profile)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to preview CSV: %s", err.Error()), http.StatusBadRequest)
			return
		}

		respJSON, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respJSON)
	})
}
