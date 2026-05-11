package docker

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/brightcolor/npc/internal/system"
)

type Container struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Image    string `json:"image"`
	PortsRaw string `json:"ports_raw"`
	Networks string `json:"networks"`
	Ports    []Port `json:"ports"`
}

type Port struct {
	HostIP        string `json:"host_ip,omitempty"`
	HostPort      int    `json:"host_port,omitempty"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
	Published     bool   `json:"published"`
}

type psLine struct {
	ID       string `json:"ID"`
	Names    string `json:"Names"`
	Image    string `json:"Image"`
	Ports    string `json:"Ports"`
	Networks string `json:"Networks"`
}

var (
	publishedPort = regexp.MustCompile(`(?:(\d+\.\d+\.\d+\.\d+|\[::\])\:)?(\d+)->(\d+)\/(tcp|udp)`)
	exposedPort   = regexp.MustCompile(`(?:^|,\s*)(\d+)\/(tcp|udp)`)
)

func Installed() bool {
	return system.Exists("docker")
}

func RunningContainers() ([]Container, error) {
	if !Installed() {
		return nil, fmt.Errorf("docker was not found")
	}
	res, err := system.Run("docker", "ps", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	return ParsePS(res.Output)
}

func ParsePS(output string) ([]Container, error) {
	var containers []Container
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw psLine
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, err
		}
		containers = append(containers, Container{
			ID: raw.ID, Name: raw.Names, Image: raw.Image, PortsRaw: raw.Ports,
			Networks: raw.Networks, Ports: ParsePorts(raw.Ports),
		})
	}
	return containers, nil
}

func ParsePorts(value string) []Port {
	seen := map[string]bool{}
	var ports []Port
	for _, match := range publishedPort.FindAllStringSubmatch(value, -1) {
		hostPort, _ := strconv.Atoi(match[2])
		containerPort, _ := strconv.Atoi(match[3])
		port := Port{HostIP: strings.Trim(match[1], "[]"), HostPort: hostPort, ContainerPort: containerPort, Protocol: match[4], Published: true}
		key := port.key()
		if !seen[key] {
			seen[key] = true
			ports = append(ports, port)
		}
	}
	for _, match := range exposedPort.FindAllStringSubmatch(value, -1) {
		containerPort, _ := strconv.Atoi(match[1])
		port := Port{ContainerPort: containerPort, Protocol: match[2]}
		key := port.key()
		if !seen[key] {
			seen[key] = true
			ports = append(ports, port)
		}
	}
	return ports
}

func (p Port) BackendHost(containerName string) string {
	if p.Published {
		return "127.0.0.1"
	}
	return containerName
}

func (p Port) BackendPort() int {
	if p.Published {
		return p.HostPort
	}
	return p.ContainerPort
}

func (p Port) Label() string {
	if p.Published {
		return fmt.Sprintf("%s:%d -> %d/%s", defaultHostIP(p.HostIP), p.HostPort, p.ContainerPort, p.Protocol)
	}
	return fmt.Sprintf("%d/%s (container network only)", p.ContainerPort, p.Protocol)
}

func (p Port) key() string {
	return fmt.Sprintf("%t:%d:%d:%s", p.Published, p.HostPort, p.ContainerPort, p.Protocol)
}

func defaultHostIP(ip string) string {
	if ip == "" {
		return "127.0.0.1"
	}
	if ip == "::" {
		return "[::]"
	}
	return ip
}
