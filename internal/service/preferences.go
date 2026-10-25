package service

import (
	"encoding/json"

	"github.com/bcrowe306/linksy/internal/models"
)

// DefaultPreferences are applied for keys a user hasn't set.
func DefaultPreferences() models.Preferences {
	return models.Preferences{
		SecurityAlerts:  true,
		TeamActivity:    true,
		WeeklyReport:    false,
		UsageAlerts:     true,
		ProductUpdates:  false,
		DefaultTimezone: "",
	}
}

func (s *Service) GetPreferences(userID string) (models.Preferences, error) {
	p := DefaultPreferences()
	raw, err := s.repo.GetUserPreferences(userID)
	if err != nil {
		return p, err
	}
	_ = json.Unmarshal([]byte(raw), &p)
	return p, nil
}

func (s *Service) UpdatePreferences(userID string, p models.Preferences) (models.Preferences, error) {
	if len(p.DefaultTimezone) > 64 {
		p.DefaultTimezone = ""
	}
	b, err := json.Marshal(p)
	if err != nil {
		return p, err
	}
	return p, s.repo.SetUserPreferences(userID, string(b))
}
