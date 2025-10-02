package teams

import (
	"fmt"
	"sync"
	"time"
)

type Team struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	Members     []string  `json:"members"`
}

type TeamManager struct {
	teams map[string]*Team
	mutex sync.RWMutex
}

func NewTeamManager() *TeamManager {
	tm := &TeamManager{
		teams: make(map[string]*Team),
	}

	// Create default team for dev mode
	defaultTeam := &Team{
		ID:          "default-team",
		Name:        "Default Team",
		Description: "Default development team",
		CreatedAt:   time.Now(),
		Members:     []string{},
	}

	tm.teams[defaultTeam.ID] = defaultTeam
	return tm
}

func (tm *TeamManager) GetTeam(id string) (*Team, bool) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	team, exists := tm.teams[id]
	return team, exists
}

func (tm *TeamManager) ListTeams() []*Team {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	teams := make([]*Team, 0, len(tm.teams))
	for _, team := range tm.teams {
		teams = append(teams, team)
	}
	return teams
}

func (tm *TeamManager) CreateTeam(name, description string) (*Team, error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Generate simple ID from name
	id := generateTeamID(name)

	// Check if team already exists
	if _, exists := tm.teams[id]; exists {
		return nil, fmt.Errorf("team with ID '%s' already exists", id)
	}

	team := &Team{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		Members:     []string{},
	}

	tm.teams[id] = team
	return team, nil
}

func (tm *TeamManager) DeleteTeam(id string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Don't allow deletion of default team
	if id == "default-team" {
		return fmt.Errorf("cannot delete default team")
	}

	if _, exists := tm.teams[id]; !exists {
		return fmt.Errorf("team with ID '%s' not found", id)
	}

	delete(tm.teams, id)
	return nil
}

func (tm *TeamManager) AddMember(teamID, memberEmail string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	team, exists := tm.teams[teamID]
	if !exists {
		return fmt.Errorf("team with ID '%s' not found", teamID)
	}

	// Check if member already exists
	for _, member := range team.Members {
		if member == memberEmail {
			return fmt.Errorf("member '%s' already exists in team", memberEmail)
		}
	}

	team.Members = append(team.Members, memberEmail)
	return nil
}

func (tm *TeamManager) RemoveMember(teamID, memberEmail string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	team, exists := tm.teams[teamID]
	if !exists {
		return fmt.Errorf("team with ID '%s' not found", teamID)
	}

	// Find and remove member
	for i, member := range team.Members {
		if member == memberEmail {
			team.Members = append(team.Members[:i], team.Members[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("member '%s' not found in team", memberEmail)
}

func (tm *TeamManager) PrintTeams() {
	teams := tm.ListTeams()
	if len(teams) == 0 {
		fmt.Println("No teams found")
		return
	}

	fmt.Println("Teams:")
	for _, team := range teams {
		fmt.Printf("  %s (%s)\n", team.Name, team.ID)
		fmt.Printf("    Description: %s\n", team.Description)
		fmt.Printf("    Created: %s\n", team.CreatedAt.Format(time.RFC3339))
		fmt.Printf("    Members: %d\n", len(team.Members))
		if len(team.Members) > 0 {
			fmt.Printf("    Member list: %v\n", team.Members)
		}
	}
}

func generateTeamID(name string) string {
	// Simple ID generation - replace spaces with hyphens and convert to lowercase
	id := ""
	for _, r := range name {
		if r == ' ' {
			id += "-"
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				id += string(r - 'A' + 'a')
			} else {
				id += string(r)
			}
		}
	}
	return id
}
