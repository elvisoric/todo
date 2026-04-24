package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultNamespace = "default"

type Todo struct {
	ID        string     `json:"id"`
	Text      string     `json:"text"`
	Namespace string     `json:"namespace"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"created_at"`
	DoneAt    *time.Time `json:"done_at,omitempty"`
	DueAt     *time.Time `json:"due_at,omitempty"`
}

type Store struct {
	path  string
	Todos []Todo `json:"todos"`
}

func dataPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".todo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "todos.json"), nil
}

func Load() (*Store, error) {
	p, err := dataPath()
	if err != nil {
		return nil, err
	}
	s := &Store{path: p}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &s.Todos); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.Todos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func generateID(existing map[string]bool) string {
	for {
		b := make([]byte, 2) // 2 bytes = 4 hex chars
		rand.Read(b)
		id := hex.EncodeToString(b)
		if !existing[id] {
			return id
		}
	}
}

func (s *Store) existingIDs() map[string]bool {
	m := make(map[string]bool)
	for _, t := range s.Todos {
		m[t.ID] = true
	}
	return m
}

func (s *Store) Add(text, namespace string, due *time.Time) Todo {
	if namespace == "" {
		namespace = defaultNamespace
	}
	t := Todo{
		ID:        generateID(s.existingIDs()),
		Text:      text,
		Namespace: namespace,
		CreatedAt: time.Now(),
		DueAt:     due,
	}
	s.Todos = append(s.Todos, t)
	return t
}

func (s *Store) FindByID(id string) *Todo {
	id = strings.ToLower(id)
	for i := range s.Todos {
		if s.Todos[i].ID == id {
			return &s.Todos[i]
		}
	}
	return nil
}

func (s *Store) MarkDone(ids []string) (done []string, notFound []string) {
	now := time.Now()
	for _, id := range ids {
		t := s.FindByID(id)
		if t == nil {
			notFound = append(notFound, id)
			continue
		}
		t.Done = true
		t.DoneAt = &now
		done = append(done, id)
	}
	return
}

// RenameSpace renames oldName to newName, including all children.
// Returns the number of todos updated.
func (s *Store) RenameSpace(oldName, newName string) int {
	count := 0
	for i := range s.Todos {
		ns := s.Todos[i].Namespace
		if ns == oldName {
			s.Todos[i].Namespace = newName
			count++
		} else if strings.HasPrefix(ns, oldName+".") {
			s.Todos[i].Namespace = newName + ns[len(oldName):]
			count++
		}
	}
	return count
}

// MoveTodos moves todos by ID to a new space.
func (s *Store) MoveTodos(ids []string, newSpace string) (moved []string, notFound []string) {
	for _, id := range ids {
		t := s.FindByID(id)
		if t == nil {
			notFound = append(notFound, id)
			continue
		}
		t.Namespace = newSpace
		moved = append(moved, id)
	}
	return
}

func (s *Store) Delete(ids []string) (deleted []string, notFound []string) {
	toDelete := make(map[string]bool)
	for _, id := range ids {
		toDelete[strings.ToLower(id)] = true
	}
	var remaining []Todo
	for _, t := range s.Todos {
		if toDelete[t.ID] {
			deleted = append(deleted, t.ID)
			delete(toDelete, t.ID)
		} else {
			remaining = append(remaining, t)
		}
	}
	s.Todos = remaining
	for id := range toDelete {
		notFound = append(notFound, id)
	}
	return
}

func (s *Store) ActiveTodos() []Todo {
	var result []Todo
	for _, t := range s.Todos {
		if !t.Done {
			result = append(result, t)
		}
	}
	return result
}

func (s *Store) DoneTodos() []Todo {
	var result []Todo
	for _, t := range s.Todos {
		if t.Done {
			result = append(result, t)
		}
	}
	return result
}

// NamespaceTree builds a tree of namespace -> count for active todos.
type NamespaceNode struct {
	Name     string
	Count    int // direct todos in this namespace
	Total    int // todos in this namespace + all children
	Children []*NamespaceNode
}

func (s *Store) NamespaceTree() *NamespaceNode {
	root := &NamespaceNode{Name: "root"}
	counts := make(map[string]int)

	for _, t := range s.Todos {
		if t.Done {
			continue
		}
		counts[t.Namespace]++
	}

	nodes := make(map[string]*NamespaceNode)
	nodes[""] = root

	// Sort namespaces for deterministic output
	var nss []string
	for ns := range counts {
		nss = append(nss, ns)
	}
	sort.Strings(nss)

	for _, ns := range nss {
		ensureNode(nodes, ns).Count = counts[ns]
	}

	// Calculate totals bottom-up
	calcTotal(root)
	return root
}

func ensureNode(nodes map[string]*NamespaceNode, ns string) *NamespaceNode {
	if n, ok := nodes[ns]; ok {
		return n
	}
	parts := strings.Split(ns, ".")
	name := parts[len(parts)-1]
	parentNS := ""
	if len(parts) > 1 {
		parentNS = strings.Join(parts[:len(parts)-1], ".")
	}
	parent := ensureNode(nodes, parentNS)
	n := &NamespaceNode{Name: name}
	parent.Children = append(parent.Children, n)
	nodes[ns] = n
	return n
}

func calcTotal(n *NamespaceNode) int {
	n.Total = n.Count
	for _, c := range n.Children {
		n.Total += calcTotal(c)
	}
	return n.Total
}

func (s *Store) TodosByNamespace(ns string) []Todo {
	var result []Todo
	for _, t := range s.Todos {
		if !t.Done && (t.Namespace == ns || strings.HasPrefix(t.Namespace, ns+".")) {
			result = append(result, t)
		}
	}
	return result
}

// AllNamespaces returns sorted unique namespaces from active todos.
func (s *Store) AllNamespaces() []string {
	m := make(map[string]bool)
	for _, t := range s.Todos {
		if !t.Done {
			m[t.Namespace] = true
		}
	}
	var result []string
	for ns := range m {
		result = append(result, ns)
	}
	sort.Strings(result)
	return result
}

// AllIDs returns all todo IDs (for shell completion).
func (s *Store) AllIDs() []string {
	var result []string
	for _, t := range s.Todos {
		result = append(result, fmt.Sprintf("%s\t%s", t.ID, Truncate(t.Text, 40)))
	}
	return result
}

func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
