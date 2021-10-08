package cgroup

type Limits map[string]string

type Cgroup interface {
	Put(groupID string, pid int, limits Limits) error
	Remove(groupID string) error
}
