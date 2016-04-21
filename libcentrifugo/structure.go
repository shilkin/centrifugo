package libcentrifugo

import (
	"errors"
	"regexp"
	"sync"
)

// ChannelOptions represent channel specific configuration for namespace or project in a whole
type ChannelOptions struct {
	// Watch determines if message published into channel will be sent into admin channel
	Watch bool `json:"watch"`

	// Publish determines if client can publish messages into channel directly
	Publish bool `json:"publish"`

	// Anonymous determines is anonymous access (with empty user ID) allowed or not
	Anonymous bool `json:"anonymous"`

	// Presence turns on(off) presence information for channels
	Presence bool `json:"presence"`

	// HistorySize determines max amount of history messages for channel, 0 means no history for channel
	HistorySize int64 `mapstructure:"history_size" json:"history_size"`

	// HistoryLifetime determines time in seconds until expiration for history messages
	HistoryLifetime int64 `mapstructure:"history_lifetime" json:"history_lifetime"`

	// JoinLeave turns on(off) join/leave messages for channels
	JoinLeave bool `mapstructure:"join_leave" json:"join_leave"`
}

// Project represents single project that uses Centrifugo.
// Note that although Centrifugo can work with several projects
// but it's recommended to have separate Centrifugo installation
// for every project (maybe except copy of your project for development).
type Project struct {
	// Name is unique project name, used as project key for client connections and API requests
	Name ProjectKey `json:"name"`

	// Secret is a secret key for project, used to sign API requests and client connection tokens
	Secret string `json:"secret"`

	// ConnLifetime determines time until connection expire, 0 means no connection expire at all
	ConnLifetime int64 `mapstructure:"connection_lifetime" json:"connection_lifetime"`

	// Namespaces - list of namespaces for project for custom channel options
	Namespaces []Namespace `json:"namespaces"`

	// ChannelOptions - default project channel options
	ChannelOptions `mapstructure:",squash"`
}

// Namespace allows to create channels with different channel options within the Project
type Namespace struct {
	// Name is a unique namespace name in project
	Name NamespaceKey `json:"name"`

	// ChannelOptions for namespace determine channel options for channels belonging to this namespace
	ChannelOptions `mapstructure:",squash"`
}

type (
	// ProjectKey is a name of project.
	ProjectKey string
	// NamespaceKey is a name of namespace unique for project.
	NamespaceKey string
	// Channel is a string channel name, can be the same for different projects.
	Channel string
	// ChannelID is unique channel identificator in Centrifugo.
	ChannelID string
	// UserID is web application user ID as string.
	UserID string // User ID
	// ConnID is a unique connection ID.
	ConnID string
	// UID is sequential message ID
	MessageID string
)

// Structure contains project structure related fields - it allows to work with Projects and Namespaces.
type Structure struct {
	sync.RWMutex
	projectList  []Project
	projectMap   map[ProjectKey]Project
	namespaceMap map[ProjectKey]map[NamespaceKey]Namespace
}

// NewStructure allows to create fully initialized structure from a slice of Projects.
func NewStructure(pl []Project) *Structure {
	s := &Structure{
		projectList: pl,
	}
	s.initialize()
	return s
}

// initialize initializes structure fields based on its project list.
func (s *Structure) initialize() {
	s.Lock()
	defer s.Unlock()
	pm := map[ProjectKey]Project{}
	nm := map[ProjectKey]map[NamespaceKey]Namespace{}
	for _, p := range s.projectList {
		pm[p.Name] = p
		nm[p.Name] = map[NamespaceKey]Namespace{}
		for _, n := range p.Namespaces {
			nm[p.Name][n.Name] = n
		}
	}
	s.projectMap = pm
	s.namespaceMap = nm
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// validate validates structure and return error if problems found
func (s *Structure) Validate() error {
	s.Lock()
	defer s.Unlock()
	var projectNames []string
	errPrefix := "config error: "
	pattern := "^[-a-zA-Z0-9_]{2,}$"
	for _, p := range s.projectList {
		name := string(p.Name)
		match, _ := regexp.MatchString(pattern, name)
		if !match {
			return errors.New(errPrefix + "wrong project name – " + name)
		}
		if p.Secret == "" {
			return errors.New(errPrefix + "secret required for project – " + name)
		}
		if stringInSlice(name, projectNames) {
			return errors.New(errPrefix + "project name must be unique – " + name)
		}
		projectNames = append(projectNames, name)

		if p.Namespaces == nil {
			continue
		}
		var nss []string
		for _, n := range p.Namespaces {
			name := string(n.Name)
			match, _ := regexp.MatchString(pattern, name)
			if !match {
				return errors.New(errPrefix + "wrong namespace name – " + name)
			}
			if stringInSlice(name, nss) {
				return errors.New(errPrefix + "namespace name must be unique for project – " + name)
			}
			nss = append(nss, name)
		}

	}
	return nil
}

// projectByKey searches for a project with specified key in structure
func (s *Structure) projectByKey(pk ProjectKey) (Project, bool) {
	s.RLock()
	defer s.RUnlock()
	p, ok := s.projectMap[pk]
	if !ok {
		return Project{}, false
	}
	return p, true
}

// channelOpts searches for channel options for specified project key and namespace key
func (s *Structure) channelOpts(pk ProjectKey, nk NamespaceKey) (ChannelOptions, error) {
	s.RLock()
	defer s.RUnlock()
	p, exists := s.projectByKey(pk)
	if !exists {
		return ChannelOptions{}, ErrProjectNotFound
	}
	if nk == NamespaceKey("") {
		return p.ChannelOptions, nil
	} else {
		n, exists := s.namespaceMap[pk][nk]
		if !exists {
			return ChannelOptions{}, ErrNamespaceNotFound
		}
		return n.ChannelOptions, nil
	}
}
