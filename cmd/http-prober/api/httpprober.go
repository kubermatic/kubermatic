package api

type Command struct {
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}
