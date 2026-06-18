package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"server/db"
	"server/models"

	"github.com/gorilla/mux"
)

// GetDepartmentOverview retrieves department overview data for a curriculum
func GetDepartmentOverview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum ID"})
		return
	}

	query := `SELECT id, curriculum_id, vision FROM curriculum_vision WHERE curriculum_id = ? AND (status = 1 OR status IS NULL)`

	var overview models.DepartmentOverview

	err = db.DB.QueryRow(query, curriculumID).Scan(
		&overview.ID,
		&overview.CurriculumID,
		&overview.Vision,
	)

	if err == sql.ErrNoRows {
		// Return empty structure with default values
		overview = models.DepartmentOverview{
			CurriculumID: curriculumID,
			Vision:       "",
			Mission:      []models.DepartmentListItem{},
			PEOs:         []models.DepartmentListItem{},
			POs:          []models.DepartmentListItem{},
			PSOs:         []models.DepartmentListItem{},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(overview)
		return
	} else if err != nil {
		log.Println("Error querying department overview:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch department overview"})
		return
	}

	// Fetch mission, PEOs, POs, PSOs ordered by position (all now use curriculum_id FK to curriculum.id)
	overview.Mission = fetchDepartmentList(curriculumID, "curriculum_mission", "mission_text")
	overview.PEOs = fetchDepartmentList(curriculumID, "curriculum_peos", "peo_text")
	overview.POs = fetchDepartmentList(curriculumID, "curriculum_pos", "po_text")
	overview.PSOs = fetchDepartmentList(curriculumID, "curriculum_psos", "pso_text")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(overview)
}

// Helper function to fetch list items from normalized tables
// Note: Shared items are now physically replicated in the database with source tracking,
// so we only need to fetch items for this curriculum (includes both owned and shared items)
func fetchDepartmentList(curriculumID int, tableName, columnName string) []models.DepartmentListItem {
	// Only fetch items with status=1 for curriculum_mission, curriculum_peos, curriculum_pos, curriculum_psos
	statusFilter := ""
	if tableName == "curriculum_mission" || tableName == "curriculum_peos" || tableName == "curriculum_pos" || tableName == "curriculum_psos" {
		statusFilter = " AND status = 1"
	}

	query := fmt.Sprintf("SELECT id, %s, visibility, source_curriculum_id FROM %s WHERE curriculum_id = ?%s ORDER BY position", columnName, tableName, statusFilter)
	rows, err := db.DB.Query(query, curriculumID)
	if err != nil {
		log.Printf("Error fetching %s: %v\n", tableName, err)
		return []models.DepartmentListItem{}
	}
	defer rows.Close()

	items := []models.DepartmentListItem{}
	for rows.Next() {
		var item models.DepartmentListItem
		var sourceDeptID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Text, &item.Visibility, &sourceDeptID); err == nil {
			if sourceDeptID.Valid {
				item.SourceCurriculumID = int(sourceDeptID.Int64)
			}
			items = append(items, item)
		}
	}

	return items
}

// SaveDepartmentOverview creates or updates department overview data
func SaveDepartmentOverview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum ID"})
		return
	}

	var overview models.DepartmentOverview
	err = json.NewDecoder(r.Body).Decode(&overview)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	overview.CurriculumID = curriculumID

	// Fetch existing data for diff
	var oldOverview models.DepartmentOverview
	var visionID int
	fetchQuery := "SELECT id, vision FROM curriculum_vision WHERE curriculum_id = ? AND (status = 1 OR status IS NULL)"
	fetchErr := db.DB.QueryRow(fetchQuery, curriculumID).Scan(&visionID, &oldOverview.Vision)

	hasExisting := fetchErr != sql.ErrNoRows
	if hasExisting {
		oldOverview.Mission = fetchDepartmentList(curriculumID, "curriculum_mission", "mission_text")
		oldOverview.PEOs = fetchDepartmentList(curriculumID, "curriculum_peos", "peo_text")
		oldOverview.POs = fetchDepartmentList(curriculumID, "curriculum_pos", "po_text")
		oldOverview.PSOs = fetchDepartmentList(curriculumID, "curriculum_psos", "pso_text")
	}

	// Check if record exists
	var existingID int
	checkQuery := "SELECT id FROM curriculum_vision WHERE curriculum_id = ? AND (status = 1 OR status IS NULL)"
	err = db.DB.QueryRow(checkQuery, curriculumID).Scan(&existingID)

	if overview.Vision == "" {
		// If vision is empty, soft delete the record if it exists (set status=0)
		_, err := db.DB.Exec("UPDATE curriculum_vision SET status = 0 WHERE curriculum_id = ? AND (status = 1 OR status IS NULL)", curriculumID)
		if err != nil && err != sql.ErrNoRows {
			log.Println("Error soft-deleting empty vision record:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete empty vision record"})
			return
		}
		overview.ID = 0
	} else {
		switch err {
		case sql.ErrNoRows:
			// INSERT new record with status=1
			insertQuery := `INSERT INTO curriculum_vision (curriculum_id, vision, status) VALUES (?, ?, 1)`
			result, err := db.DB.Exec(insertQuery, curriculumID, overview.Vision)
			if err != nil {
				log.Println("Error inserting department overview:", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save department overview"})
				return
			}
			id, _ := result.LastInsertId()
			overview.ID = int(id)
		case nil:
			// UPDATE existing record and set status=1
			updateQuery := `UPDATE curriculum_vision SET vision = ?, status = 1 WHERE curriculum_id = ? AND (status = 1 OR status IS NULL)`
			_, err := db.DB.Exec(updateQuery, overview.Vision, curriculumID)
			if err != nil {
				log.Println("Error updating department overview:", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update department overview"})
				return
			}
			overview.ID = existingID
		default:
			log.Println("Error checking existing record:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save department overview"})
			return
		}
	}

	// Save list items to normalized tables (all now use curriculum_id FK to curriculum.id)
	saveDepartmentList(curriculumID, "curriculum_mission", "mission_text", overview.Mission)
	saveDepartmentList(curriculumID, "curriculum_peos", "peo_text", overview.PEOs)
	saveDepartmentList(curriculumID, "curriculum_pos", "po_text", overview.POs)
	saveDepartmentList(curriculumID, "curriculum_psos", "pso_text", overview.PSOs)

	// Generate granular diff and log the activity
	if hasExisting {
		// Vision change
		if oldOverview.Vision != overview.Vision {
			diff := map[string]map[string]interface{}{
				"vision": {"old": oldOverview.Vision, "new": overview.Vision},
			}
			LogCurriculumActivityWithDiff(curriculumID, "Vision Updated",
				"Updated department vision", "System", diff)
		}

		// Mission changes (per index)
		detectArrayChanges(curriculumID, "Mission", oldOverview.Mission, overview.Mission)

		// PEO changes (per index)
		detectArrayChanges(curriculumID, "PEO", oldOverview.PEOs, overview.PEOs)

		// PO changes (per index)
		detectArrayChanges(curriculumID, "PO", oldOverview.POs, overview.POs)

		// PSO changes (per index)
		detectArrayChanges(curriculumID, "PSO", oldOverview.PSOs, overview.PSOs)
	} else {
		LogCurriculumActivity(curriculumID, "Department Overview Created",
			"Created department vision, mission, PEOs, POs, and PSOs", "System")
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(overview)
}

// Helper function to save list items to normalized tables with proper update/insert/delete logic
func saveDepartmentList(curriculumID int, tableName, columnName string, items []models.DepartmentListItem) error {
	log.Printf("Saving %d items to %s for curriculum %d\n", len(items), tableName, curriculumID)

	// Fetch existing ACTIVE items from database (status = 1 or NULL)
	// Soft-deleted items (status = 0) are kept for history but excluded from active operations
	existingItems := make(map[int]struct {
		text               string
		visibility         string
		sourceDepartmentID sql.NullInt64
	})

	statusFilter := ""
	if tableName == "curriculum_mission" || tableName == "curriculum_peos" || tableName == "curriculum_pos" || tableName == "curriculum_psos" {
		statusFilter = " AND (status = 1 OR status IS NULL)"
	}

	query := fmt.Sprintf("SELECT id, %s, visibility, source_curriculum_id FROM %s WHERE curriculum_id = ?%s", columnName, tableName, statusFilter)
	rows, err := db.DB.Query(query, curriculumID)
	if err != nil {
		log.Printf("Error fetching existing items: %v\n", err)
		return err
	}

	for rows.Next() {
		var id int
		var text, visibility string
		var sourceDeptID sql.NullInt64
		if err := rows.Scan(&id, &text, &visibility, &sourceDeptID); err == nil {
			existingItems[id] = struct {
				text               string
				visibility         string
				sourceDepartmentID sql.NullInt64
			}{text, visibility, sourceDeptID}
		}
	}
	rows.Close()

	// Track which IDs are still present
	presentIDs := make(map[int]bool)

	// Process each item in the input
	for i, item := range items {
		if item.Text == "" {
			continue
		}

		if item.ID > 0 {
			// Existing item - check if it needs updating
			presentIDs[item.ID] = true
			existing, exists := existingItems[item.ID]

			if !exists {
				log.Printf("Warning: Item ID %d not found in database\n", item.ID)
				continue
			}

			// Check if text changed
			textChanged := existing.text != item.Text

			// Handle shared items from other curricula
			if existing.sourceDepartmentID.Valid && existing.sourceDepartmentID.Int64 != int64(curriculumID) {
				// This is a shared item from another curriculum
				if textChanged {
					// User is modifying a shared item - make it their own
					sourceCurrID := int(existing.sourceDepartmentID.Int64)
					log.Printf("Item %d from curriculum %d is being modified - converting to owned item\n",
						item.ID, sourceCurrID)

					// Determine item type for tracking table
					itemType := ""
					switch tableName {
					case "curriculum_mission":
						itemType = "mission"
					case "curriculum_peos":
						itemType = "peos"
					case "curriculum_pos":
						itemType = "pos"
					case "curriculum_psos":
						itemType = "psos"
					}

					// STEP 1: Remove this item from the sharing tracking table
					// This prevents it from being deleted when we unshare the original
					_, err := db.DB.Exec(`
						DELETE FROM sharing_tracking 
						WHERE target_curriculum_id = ? AND copied_item_id = ? AND item_type = ?
					`, curriculumID, item.ID, itemType)
					if err != nil {
						log.Printf("Error removing from tracking table: %v\n", err)
					}

					// STEP 2: Update this item to be owned by this curriculum with the new text
					visibility := "UNIQUE"

					updateQuery := fmt.Sprintf("UPDATE %s SET %s = ?, visibility = ?, position = ?, source_curriculum_id = NULL WHERE id = ?", tableName, columnName)
					_, err = db.DB.Exec(updateQuery, item.Text, visibility, i, item.ID)
					if err != nil {
						log.Printf("Error converting shared item to owned: %v\n", err)
						return err
					}

					// STEP 3: Find and unshare the original item in the source curriculum
					// Now when we unshare, this item won't be deleted since it's no longer in tracking
					var originalItemID int
					findOriginalQuery := fmt.Sprintf(`
							SELECT id FROM %s 
							WHERE curriculum_id = ? 
							AND %s = ? 
							AND (source_curriculum_id IS NULL OR source_curriculum_id = ?)
							AND visibility = 'CLUSTER'
							LIMIT 1
						`, tableName, columnName)

					err = db.DB.QueryRow(findOriginalQuery, sourceCurrID, existing.text, sourceCurrID).Scan(&originalItemID)
					if err == nil {
						// Found the original item - unshare it from remaining curricula
						log.Printf("Unsharing original item %d in source curriculum %d\n", originalItemID, sourceCurrID)
						if err := unshareItemFromCluster(sourceCurrID, originalItemID, tableName); err != nil {
							log.Printf("Error unsharing original item: %v\n", err)
						}

						// Update the original to UNIQUE visibility
						updateOriginalQuery := fmt.Sprintf("UPDATE %s SET visibility = 'UNIQUE' WHERE id = ?", tableName)
						db.DB.Exec(updateOriginalQuery, originalItemID)
					} else {
						log.Printf("Could not find original item in source curriculum: %v\n", err)
					}
				} else {
					// Text not changed, just update position
					updateQuery := fmt.Sprintf("UPDATE %s SET position = ? WHERE id = ?", tableName)
					_, err := db.DB.Exec(updateQuery, i, item.ID)
					if err != nil {
						log.Printf("Error updating position: %v\n", err)
						return err
					}
				}
				continue
			}

			// Item is owned by this curriculum - allow modification
			if textChanged && existing.visibility == "CLUSTER" {
				// Text was modified on a shared item - update all shared copies with the new text
				log.Printf("Item %d was modified and is currently shared - updating all shared copies\n", item.ID)

				// Update all shared copies in other curricula
				if err := updateSharedItemInCluster(curriculumID, item.ID, tableName, columnName, item.Text); err != nil {
					log.Printf("Error updating shared items in cluster: %v\n", err)
				}

				// Keep visibility as CLUSTER (still shared)
				item.Visibility = "CLUSTER"
			}

			// Update the item and ensure status=1 for soft-delete tables
			visibility := item.Visibility
			if visibility == "" {
				visibility = existing.visibility // Preserve existing visibility if not specified
			}

			if tableName == "curriculum_mission" || tableName == "curriculum_peos" || tableName == "curriculum_pos" || tableName == "curriculum_psos" {
				updateQuery := fmt.Sprintf("UPDATE %s SET %s = ?, visibility = ?, position = ?, status = 1 WHERE id = ?", tableName, columnName)
				_, err := db.DB.Exec(updateQuery, item.Text, visibility, i, item.ID)
				if err != nil {
					log.Printf("Error updating item %d: %v\n", item.ID, err)
					return err
				}
			} else {
				updateQuery := fmt.Sprintf("UPDATE %s SET %s = ?, visibility = ?, position = ? WHERE id = ?", tableName, columnName)
				_, err := db.DB.Exec(updateQuery, item.Text, visibility, i, item.ID)
				if err != nil {
					log.Printf("Error updating item %d: %v\n", item.ID, err)
					return err
				}
			}
		} else {
			// New item - insert it with status=1
			visibility := item.Visibility
			if visibility == "" {
				visibility = "UNIQUE"
			}

			// Set status=1 for tables that support soft delete
			if tableName == "curriculum_mission" || tableName == "curriculum_peos" || tableName == "curriculum_pos" || tableName == "curriculum_psos" {
				insertQuery := fmt.Sprintf(`
				INSERT INTO %s (curriculum_id, %s, visibility, position, source_curriculum_id, status) 
					VALUES (?, ?, ?, ?, NULL, 1)
				`, tableName, columnName)
				log.Printf("Inserting new item into %s at position %d with status=1\n", tableName, i)
				_, err := db.DB.Exec(insertQuery, curriculumID, item.Text, visibility, i)
				if err != nil {
					log.Printf("Error inserting new item into %s: %v\n", tableName, err)
					return err
				}
				log.Printf("Successfully inserted new item into %s at position %d\n", tableName, i)
			} else {
				insertQuery := fmt.Sprintf(`
				INSERT INTO %s (curriculum_id, %s, visibility, position, source_curriculum_id) 
					VALUES (?, ?, ?, ?, NULL)
				`, tableName, columnName)
				_, err := db.DB.Exec(insertQuery, curriculumID, item.Text, visibility, i)
				if err != nil {
					log.Printf("Error inserting new item: %v\n", err)
					return err
				}
			}
		}
	}

	// Delete items that are no longer present
	for id, existing := range existingItems {
		if !presentIDs[id] {
			// Item was removed from the list

			// Check if this is owned by this curriculum
			isOwnedItem := !existing.sourceDepartmentID.Valid || existing.sourceDepartmentID.Int64 == int64(curriculumID)

			if isOwnedItem {
				// Owned item being deleted
				// If it was shared (CLUSTER), unshare it first
				if existing.visibility == "CLUSTER" {
					log.Printf("Deleting owned item %d which was shared - unsharing first\n", id)
					if err := unshareItemFromCluster(curriculumID, id, tableName); err != nil {
						log.Printf("Error unsharing deleted item: %v\n", err)
					}
				}
			} else {
				// Shared item from another curriculum being deleted
				log.Printf("Deleting shared item %d from curriculum %d\n", id, existing.sourceDepartmentID.Int64)
			}

			// SOFT DELETE for curriculum_mission, curriculum_peos, curriculum_pos, curriculum_psos: set status=0 instead of deleting
			if tableName == "curriculum_mission" || tableName == "curriculum_peos" || tableName == "curriculum_pos" || tableName == "curriculum_psos" {
				softDeleteQuery := fmt.Sprintf("UPDATE %s SET status = 0 WHERE id = ?", tableName)
				_, err := db.DB.Exec(softDeleteQuery, id)
				if err != nil {
					log.Printf("Error soft-deleting %s %d: %v\\n", tableName, id, err)
				}
			} else {
				// Delete the item (works for both owned and shared items)
				deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
				_, err := db.DB.Exec(deleteQuery, id)
				if err != nil {
					log.Printf("Error deleting item %d: %v\\n", id, err)
				}
			}
		}
	}

	return nil
}

// Helper function to compare DepartmentListItem arrays
func listItemArraysEqual(a, b []models.DepartmentListItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// detectArrayChanges compares two DepartmentListItem arrays and logs individual item changes
func detectArrayChanges(regulationID int, label string, oldArray, newArray []models.DepartmentListItem) {
	maxLen := len(oldArray)
	if len(newArray) > maxLen {
		maxLen = len(newArray)
	}

	for i := 0; i < maxLen; i++ {
		var oldVal, newVal string
		if i < len(oldArray) {
			oldVal = oldArray[i].Text
		}
		if i < len(newArray) {
			newVal = newArray[i].Text
		}

		// Detect changes
		if oldVal != newVal {
			diff := map[string]map[string]interface{}{
				fmt.Sprintf("%s[%d]", label, i): {"old": oldVal, "new": newVal},
			}

			var action, description string
			if oldVal == "" {
				action = fmt.Sprintf("%s[%d] Added", label, i)
				description = fmt.Sprintf("Added %s item at index %d", label, i)
			} else if newVal == "" {
				action = fmt.Sprintf("%s[%d] Deleted", label, i)
				description = fmt.Sprintf("Deleted %s item at index %d", label, i)
			} else {
				action = fmt.Sprintf("%s[%d] Updated", label, i)
				description = fmt.Sprintf("Updated %s item at index %d", label, i)
			}

			LogCurriculumActivityWithDiff(regulationID, action, description, "System", diff)
		}
	}
}
