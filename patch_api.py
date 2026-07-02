import re

with open("backend/cmd/api/main.go", "r") as f:
    content = f.read()

# Add variables for signals
content = content.replace("var cOpps, cGaps, cTrends, cRuns int64", "var cOpps, cGaps, cTrends, cRuns, cSignals int64")
content = content.replace("db.QueryRowContext(r.Context(), \"SELECT COUNT(*) FROM import_gaps\").Scan(&cGaps)", "db.QueryRowContext(r.Context(), \"SELECT COUNT(*) FROM import_gaps\").Scan(&cGaps)\n\tdb.QueryRowContext(r.Context(), \"SELECT COUNT(*) FROM signals WHERE status='new'\").Scan(&cSignals)")

# Add to json encode
content = content.replace("\"import_gaps\":     cGaps,", "\"import_gaps\":     cGaps,\n\t\t\t\"active_signals\":  cSignals,")

# Add publications_by_year
publications_by_year_logic = """
		var pubByYear []map[string]interface{}
		rowsPubs, err := db.QueryContext(r.Context(), "SELECT publication_year, COUNT(*) FROM publications WHERE publication_year IS NOT NULL AND publication_year >= 2010 GROUP BY publication_year ORDER BY publication_year")
		if err == nil {
			defer rowsPubs.Close()
			for rowsPubs.Next() {
				var year, count int
				rowsPubs.Scan(&year, &count)
				pubByYear = append(pubByYear, map[string]interface{}{"year": year, "count": count})
			}
		}
"""
content = content.replace("var lastCollected *string", publications_by_year_logic + "\n\t\tvar lastCollected *string")
content = content.replace("\"collector_runs\":  cRuns,", "\"collector_runs\":  cRuns,\n\t\t\t\"publications_by_year\": pubByYear,")

with open("backend/cmd/api/main.go", "w") as f:
    f.write(content)
