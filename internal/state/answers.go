package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AnswersDirName is the subdirectory of the state dir holding the answers file.
// It is a directory, not a sibling <name>.json, because Store.GetAll parses
// every *.json directly in the state dir as a module state (and engines older
// than the answers file do too): a sibling file would read as a phantom module.
const AnswersDirName = "answers"

// answersFileName is the answers file inside AnswersDirName.
const answersFileName = "answers.json"

// Answers is the per-machine record of prompt answers given interactively,
// keyed module -> prompt key -> value. Only answers the user was actually asked
// are recorded (never defaults), so config still overrides them and a changed
// module default still reaches a machine that never chose. Engine-owned: it is
// not meant to be hand-edited.
type Answers struct {
	path    string
	answers map[string]map[string]string
}

type answersFile struct {
	SchemaVersion int                          `json:"schema_version"`
	Answers       map[string]map[string]string `json:"answers"`
}

// AnswersPath returns the answers file path for a state dir.
func AnswersPath(stateDir string) string {
	return filepath.Join(stateDir, AnswersDirName, answersFileName)
}

// LoadAnswers reads the answers file under stateDir. An absent file is an empty
// set of answers. A present but malformed file is an error: treating it as empty
// would silently swap the user's choices for module defaults on the next run.
func LoadAnswers(stateDir string) (*Answers, error) {
	a := &Answers{path: AnswersPath(stateDir), answers: map[string]map[string]string{}}
	data, err := os.ReadFile(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return nil, err
	}
	var f answersFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", a.path, err)
	}
	for mod, keys := range f.Answers {
		if len(keys) > 0 {
			a.answers[mod] = keys
		}
	}
	return a, nil
}

// NewAnswers returns an empty in-memory answers set that saves to the answers
// file under stateDir.
func NewAnswers(stateDir string) *Answers {
	return &Answers{path: AnswersPath(stateDir), answers: map[string]map[string]string{}}
}

// Path returns the file the answers load from and save to.
func (a *Answers) Path() string { return a.path }

// Get returns the saved answer for module.key, if any.
func (a *Answers) Get(module, key string) (string, bool) {
	v, ok := a.answers[module][key]
	return v, ok
}

// Set records an answer in memory; call Save to persist it.
func (a *Answers) Set(module, key, value string) {
	if a.answers[module] == nil {
		a.answers[module] = map[string]string{}
	}
	a.answers[module][key] = value
}

// Save writes the answers atomically (temp file + rename), creating the
// directory if needed.
func (a *Answers) Save() error {
	if err := os.MkdirAll(filepath.Dir(a.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(answersFile{SchemaVersion: 1, Answers: a.answers}, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(a.path), ".answers-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), a.path)
}
