package server

import (
	"net/http"
	"sort"
)

type TemplateResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

func (s *Server) ListTemplates(w http.ResponseWriter, r *http.Request) {
	if s.templateRegistry == nil {
		writeJSON(w, http.StatusOK, []TemplateResponse{})
		return
	}

	templates := s.templateRegistry.List()
	responses := make([]TemplateResponse, 0, len(templates))
	for _, tmpl := range templates {
		responses = append(responses, TemplateResponse{
			ID:          tmpl.Config.ID,
			Name:        tmpl.Config.Name,
			Description: tmpl.Config.Description,
			Category:    tmpl.Config.Category,
		})
	}

	sort.Slice(responses, func(i, j int) bool {
		return responses[i].ID < responses[j].ID
	})

	writeJSON(w, http.StatusOK, responses)
}
