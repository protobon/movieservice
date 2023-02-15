package memory

import (
	"context"
	"errors"
	"movieservice/pkg/discovery"
	"sync"
	"time"
)

type serviceName string

type instanceID string

// Registry defines an in-memory service registry.
type Registry struct {
	sync.RWMutex
	serviceAddresses map[serviceName]map[instanceID]*serviceInstance
}

type serviceInstance struct {
	hostPort   string
	lastActive time.Time
}

// NewRegistry creates a new in-memory service registry instance.
func NewRegistry() *Registry {
	return &Registry{serviceAddresses: map[serviceName]map[instanceID]*serviceInstance{}}
}

func (r *Registry) Register(ctx context.Context, instance instanceID,
	service serviceName, hostPort string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.serviceAddresses[service]; !ok {
		r.serviceAddresses[service] = map[instanceID]*serviceInstance{}
	}

	r.serviceAddresses[service][instance] = &serviceInstance{hostPort: hostPort, lastActive: time.Now()}
	return nil
}

// Deregister removes a service record from the registry.
func (r *Registry) Deregister(ctx context.Context, instance instanceID, service serviceName) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.serviceAddresses[service]; !ok {
		return nil
	}
	delete(r.serviceAddresses[service], instance)
	return nil
}

// ReportHealthyState is a push mechanism for
// reporting healthy state to the registry.
func (r *Registry) ReportHealthyState(instance instanceID, service serviceName) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.serviceAddresses[service]; !ok {
		return errors.New("service is not registered yet")
	}
	if _, ok := r.serviceAddresses[service][instance]; !ok {
		return errors.New("service instance is not registered yet")
	}
	r.serviceAddresses[service][instance].lastActive = time.Now()
	return nil
}

// ServiceAddresses returns the list of addresses of
// active instances of the given service.
func (r *Registry) ServiceAddresses(ctx context.Context, service serviceName) ([]string, error) {
	r.RLock()
	defer r.Unlock()

	if len(r.serviceAddresses[service]) == 0 {
		return nil, discovery.ErrNotFound
	}

	var res []string
	for _, i := range r.serviceAddresses[service] {
		if i.lastActive.Before(time.Now().Add(-5 * time.Second)) {
			continue
		}
		res = append(res, i.hostPort)
	}

	return res, nil
}
