package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lib/pq"
)

// ── Mapeamento área de pesquisa → setores relevantes ────────────────────────

var areaSectorMap = map[string][]string{
	"Ciências Agrárias": {
		"agro", "agric", "defensivo", "fertilizante", "semente", "pecuária",
		"alimento", "florestal", "celulose", "madeira", "irrigação", "pesquisa agropecuária",
	},
	"Ciências Biológicas": {
		"biotecnologia", "farmacêutica", "farmaceutica", "controle biológico", "biológico",
		"saúde animal", "diagnóstico", "pesquisa agropecuária",
	},
	"Engenharias": {
		"máquinas agrícolas", "maquinas agricolas", "automação", "automacao",
		"irrigação", "irrigacao", "energia", "construção", "construcao",
	},
	"Ciências Exatas": {
		"software", "tecnologia", "agritech", "inteligência artificial", "ia",
		"dados", "gestão", "gestao",
	},
}

// lineKeywords extrai palavras relevantes das linhas de pesquisa
func lineKeywords(lines []string) []string {
	stopWords := map[string]bool{
		"de": true, "do": true, "da": true, "e": true, "em": true,
		"ao": true, "na": true, "no": true, "para": true, "com": true,
	}
	var words []string
	for _, line := range lines {
		for _, w := range strings.Fields(strings.ToLower(line)) {
			w = strings.Trim(w, ".,;()[]")
			if len(w) >= 4 && !stopWords[w] {
				words = append(words, w)
			}
		}
	}
	return words
}

func computeMatchScore(mainArea string, researchLines []string, partnerSector string) (float64, []string) {
	sector := strings.ToLower(partnerSector)
	score := 0.0
	var reasons []string

	// 1. Afinidade área → setor (peso 0.5)
	for _, kw := range areaSectorMap[mainArea] {
		if strings.Contains(sector, kw) {
			score += 0.5
			reasons = append(reasons, "Setor \""+partnerSector+"\" alinhado com "+mainArea)
			break
		}
	}

	// 2. Overlap entre linhas de pesquisa e setor do parceiro (peso 0.1 por match, máx 0.5)
	lineKws := lineKeywords(researchLines)
	matched := map[string]bool{}
	for _, lkw := range lineKws {
		if matched[lkw] {
			continue
		}
		if strings.Contains(sector, lkw) {
			score += 0.1
			matched[lkw] = true
			if len(reasons) == 0 || !strings.HasPrefix(reasons[len(reasons)-1], "Linha") {
				reasons = append(reasons, "Linha de pesquisa compatível com setor")
			}
		}
		if score >= 1.0 {
			break
		}
	}

	if score > 1.0 {
		score = 1.0
	}
	return score, reasons
}

// ── GET /api/v1/departments ──────────────────────────────────────────────────

type DeptGroup struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	ResearchLines []string `json:"research_lines"`
	MainArea      string   `json:"main_area"`
}

type Department struct {
	Code   string      `json:"code"`
	Groups []DeptGroup `json:"groups"`
}

func departmentsHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(),
			`SELECT id, name, department, research_lines, main_area
			 FROM research_groups ORDER BY department, name`)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		byDept := map[string]*Department{}
		var order []string

		for rows.Next() {
			var id int64
			var name, dept, mainArea string
			var lines []string
			if err := rows.Scan(&id, &name, &dept, pq.Array(&lines), &mainArea); err != nil {
				continue
			}

			if _, ok := byDept[dept]; !ok {
				byDept[dept] = &Department{Code: dept}
				order = append(order, dept)
			}
			byDept[dept].Groups = append(byDept[dept].Groups, DeptGroup{
				ID: id, Name: name, ResearchLines: lines, MainArea: mainArea,
			})
		}

		depts := make([]Department, 0, len(order))
		for _, code := range order {
			depts = append(depts, *byDept[code])
		}
		writeJSON(w, http.StatusOK, depts)
	})
}

// ── POST /api/v1/matchmaking/compute ────────────────────────────────────────

func computeMatchmakingHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "POST only")
			return
		}
		ctx := r.Context()

		// Load all groups
		gRows, err := db.QueryContext(ctx,
			`SELECT id, research_lines, main_area FROM research_groups`)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer gRows.Close()

		type group struct {
			ID            int64
			ResearchLines []string
			MainArea      string
		}
		var groups []group
		for gRows.Next() {
			var g group
			if err := gRows.Scan(&g.ID, pq.Array(&g.ResearchLines), &g.MainArea); err != nil {
				continue
			}
			groups = append(groups, g)
		}
		gRows.Close()

		// Load all company partners
		pRows, err := db.QueryContext(ctx,
			`SELECT id, sector FROM partners WHERE partner_type = 'empresa'`)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer pRows.Close()

		type partner struct {
			ID     int64
			Sector string
		}
		var partners []partner
		for pRows.Next() {
			var p partner
			var sector sql.NullString
			if err := pRows.Scan(&p.ID, &sector); err != nil {
				continue
			}
			p.Sector = sector.String
			partners = append(partners, p)
		}
		pRows.Close()

		// Compute and upsert matches
		inserted := 0
		for _, g := range groups {
			for _, p := range partners {
				score, reasons := computeMatchScore(g.MainArea, g.ResearchLines, p.Sector)
				if score < 0.1 {
					continue // skip irrelevant
				}
				reasonsJSON, _ := json.Marshal(reasons)
				_, err := db.ExecContext(ctx, `
					INSERT INTO matches (group_id, partner_id, score, reasons)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (group_id, partner_id) DO UPDATE
					  SET score = EXCLUDED.score, reasons = EXCLUDED.reasons, updated_at = NOW()`,
					g.ID, p.ID, score, string(reasonsJSON))
				if err == nil {
					inserted++
				}
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"computed": inserted,
			"groups":   len(groups),
			"partners": len(partners),
		})
	})
}

// ── GET /api/v1/matchmaking ──────────────────────────────────────────────────

func matchmakingHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		minScore := 0.1
		rows, err := db.QueryContext(r.Context(), `
			SELECT
				m.id, m.score, m.reasons, m.status,
				g.id, g.name, g.department, g.main_area,
				p.id, p.name, p.sector, p.location
			FROM matches m
			JOIN research_groups g ON g.id = m.group_id
			JOIN partners p        ON p.id = m.partner_id
			WHERE m.score >= $1
			ORDER BY m.score DESC
			LIMIT 200`, minScore)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type matchRow struct {
			ID          int64    `json:"id"`
			Score       float64  `json:"score"`
			Reasons     []string `json:"reasons"`
			Status      string   `json:"status"`
			GroupID     int64    `json:"group_id"`
			GroupName   string   `json:"group_name"`
			Department  string   `json:"department"`
			MainArea    string   `json:"main_area"`
			PartnerID   int64    `json:"partner_id"`
			PartnerName string   `json:"partner_name"`
			Sector      string   `json:"sector"`
			Location    string   `json:"location"`
		}

		var results []matchRow
		for rows.Next() {
			var m matchRow
			var reasonsRaw []byte
			var loc sql.NullString
			err := rows.Scan(
				&m.ID, &m.Score, &reasonsRaw, &m.Status,
				&m.GroupID, &m.GroupName, &m.Department, &m.MainArea,
				&m.PartnerID, &m.PartnerName, &m.Sector, &loc,
			)
			if err != nil {
				continue
			}
			m.Location = loc.String
			json.Unmarshal(reasonsRaw, &m.Reasons)
			results = append(results, m)
		}
		if results == nil {
			results = []matchRow{}
		}
		writeJSON(w, http.StatusOK, results)
	})
}

// ── PATCH /api/v1/matchmaking/{id} ──────────────────────────────────────────

func patchMatchHandler(db *sql.DB) http.HandlerFunc {
	return corsWrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			writeErr(w, http.StatusMethodNotAllowed, "PATCH only")
			return
		}
		id := r.PathValue("id")
		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		allowed := map[string]bool{"pending": true, "contacted": true, "in_progress": true, "closed": true}
		if !allowed[body.Status] {
			writeErr(w, http.StatusBadRequest, "invalid status")
			return
		}
		_, err := db.ExecContext(r.Context(),
			`UPDATE matches SET status=$1, updated_at=NOW() WHERE id=$2`, body.Status, id)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": body.Status})
	})
}
