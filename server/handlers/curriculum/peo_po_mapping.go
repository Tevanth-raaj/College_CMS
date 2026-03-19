package curriculum

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"strconv"

	"github.com/gorilla/mux"
)

// GetPEOPOMapping handles GET /curriculum/:id/peo-po-mapping
func GetPEOPOMapping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid curriculum ID", http.StatusBadRequest)
		return
	}

	// Fetch PEO-PO mappings from existing table
	poMatrix := make(map[string]int)
	poRows, err := db.DB.Query(`
		SELECT peo_index, po_index, mapping_value
		FROM peo_po_mapping
		WHERE curriculum_id = ?
	`, curriculumID)
	if err != nil {
		log.Printf("Error fetching PEO-PO mappings for curriculum %d: %v", curriculumID, err)
		// Don't return error, table might not exist yet or there might be a temporary query issue
	} else {
		defer poRows.Close()
		rowCount := 0
		for poRows.Next() {
			var peoIndex, poIndex, mappingValue int
			if err := poRows.Scan(&peoIndex, &poIndex, &mappingValue); err != nil {
				log.Println("Error scanning PEO-PO mapping:", err)
				continue
			}
			// Convert to 0-based indexing for frontend
			key := fmt.Sprintf("%d-%d", peoIndex-1, poIndex-1)
			poMatrix[key] = mappingValue
			log.Printf("PEO-PO Mapping loaded: curriculum=%d, key=%s, value=%d", curriculumID, key, mappingValue)
			rowCount++
		}
		log.Printf("Loaded %d PEO-PO mappings for curriculum %d", rowCount, curriculumID)
	}

	// Fetch PSO-PO mappings from pso_po_mapping table
	psoPoMatrix := make(map[string]int)
	psoPoRows, err := db.DB.Query(`
		SELECT pso_index, po_index, mapping_value
		FROM pso_po_mapping
		WHERE curriculum_id = ?
	`, curriculumID)
	if err != nil {
		log.Println("Error fetching PSO-PO mappings (table may not exist yet):", err)
		// Don't return error, table might not exist yet
	} else {
		defer psoPoRows.Close()
		for psoPoRows.Next() {
			var psoIndex, poIndex, mappingValue int
			if err := psoPoRows.Scan(&psoIndex, &poIndex, &mappingValue); err != nil {
				log.Println("Error scanning PSO-PO mapping:", err)
				continue
			}
			// Convert to 0-based indexing for frontend
			key := fmt.Sprintf("%d-%d", psoIndex-1, poIndex-1)
			psoPoMatrix[key] = mappingValue
		}
	}

	response := models.PEOPOMappingResponse{
		PoMatrix:    poMatrix,
		PsoPoMatrix: psoPoMatrix,
	}

	json.NewEncoder(w).Encode(response)
}

// SavePEOPOMapping handles POST /curriculum/:id/peo-po-mapping
func SavePEOPOMapping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid curriculum ID", http.StatusBadRequest)
		return
	}

	var request models.PEOPOMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Error decoding request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("SavePEOPOMapping: curriculum_id=%d, mappings count=%d, psoPoMappings count=%d",
		curriculumID, len(request.Mappings), len(request.PSOPOMappings))

	// Start transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		http.Error(w, "Failed to save mappings", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete existing PO mappings from existing table
	_, err = tx.Exec("DELETE FROM peo_po_mapping WHERE curriculum_id = ?", curriculumID)
	if err != nil {
		log.Println("Error deleting existing PO mappings:", err)
		http.Error(w, "Failed to save mappings", http.StatusInternalServerError)
		return
	}

	// Delete existing PSO-PO mappings
	_, err = tx.Exec("DELETE FROM pso_po_mapping WHERE curriculum_id = ?", curriculumID)
	if err != nil {
		log.Println("Error deleting existing PSO-PO mappings (table may not exist yet):", err)
		// Don't fail if table doesn't exist yet
	}

	// Insert new PEO-PO mappings
	for _, mapping := range request.Mappings {
		log.Printf("Inserting PEO-PO mapping: curriculum=%d, peo_index=%d, po_index=%d, value=%d",
			curriculumID, mapping.PEOIndex, mapping.POIndex, mapping.MappingValue)
		_, err = tx.Exec(`
			INSERT INTO peo_po_mapping (curriculum_id, peo_index, po_index, mapping_value)
			VALUES (?, ?, ?, ?)
		`, curriculumID, mapping.PEOIndex, mapping.POIndex, mapping.MappingValue)
		if err != nil {
			log.Println("Error inserting PEO-PO mapping:", err)
			http.Error(w, "Failed to save PEO-PO mappings", http.StatusInternalServerError)
			return
		}
	}

	// Insert PSO-PO mappings
	for _, mapping := range request.PSOPOMappings {
		_, err = tx.Exec(`
			INSERT INTO pso_po_mapping (curriculum_id, pso_index, po_index, mapping_value)
			VALUES (?, ?, ?, ?)
		`, curriculumID, mapping.PSOIndex, mapping.POIndex, mapping.MappingValue)
		if err != nil {
			log.Println("Error inserting PSO-PO mapping:", err)
			http.Error(w, "Failed to save PSO-PO mappings", http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		log.Println("Error committing transaction:", err)
		http.Error(w, "Failed to save mappings", http.StatusInternalServerError)
		return
	}

	// Log the activity
	LogCurriculumActivity(curriculumID, "PEO-PO & PSO-PO Mapping Saved",
		"Updated PEO-PO and PSO-PO mappings for the curriculum", "System")

	json.NewEncoder(w).Encode(map[string]string{"message": "Mappings saved successfully"})
}
