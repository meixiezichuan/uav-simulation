package main

type ServiceType string
type Properties map[string]string

const (
	ServiceTypeInfo   ServiceType = "information"
	ServiceTypeDevice ServiceType = "device"
)

type ServiceMeta struct {
	Address    string // access method: url/ip:port/etc, unique
	Type       ServiceType
	Properties Properties
}

type HealthChecks struct {
	Type string // Check type: http/ttl/tcp/udp/script/etc

	Interval    string // from definition
	Timeout     string // from definition
	URL         string // url when type is http
	ExposedPort int    // port used if type is tcp/udp
	ScriptPath  string // if type is script, the path should be specified
}

// ServiceAgent presents service information to notify local agent
type ServiceAgent struct {
	ServiceMeta
	Name   string
	Checks HealthChecks
}

// ServiceRegistration presents information of service to register
type ServiceRegistration struct {
	ServiceMeta
	Name string
	Node string
}

// ServiceCatalog presents the instances of service to share
type ServiceCatalog struct {
	Name      string
	Instances []ServiceMeta
}
