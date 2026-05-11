package docker

import "testing"

func TestParsePorts(t *testing.T) {
	ports := ParsePorts("0.0.0.0:8080->80/tcp, [::]:8443->443/tcp, 9000/udp")
	if len(ports) != 3 {
		t.Fatalf("expected 3 ports, got %d: %#v", len(ports), ports)
	}
	if !ports[0].Published || ports[0].HostPort != 8080 || ports[0].ContainerPort != 80 {
		t.Fatalf("unexpected first port: %#v", ports[0])
	}
	if ports[2].Published || ports[2].ContainerPort != 9000 || ports[2].Protocol != "udp" {
		t.Fatalf("unexpected exposed port: %#v", ports[2])
	}
}

func TestParsePS(t *testing.T) {
	output := `{"ID":"abc","Image":"nginx","Names":"web","Ports":"0.0.0.0:8080->80/tcp","Networks":"bridge"}`
	containers, err := ParsePS(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 || containers[0].Name != "web" || containers[0].Ports[0].BackendPort() != 8080 {
		t.Fatalf("unexpected containers: %#v", containers)
	}
}
