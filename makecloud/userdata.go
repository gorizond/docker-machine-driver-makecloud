package makecloud

import (
	"fmt"
	"os"
	"strings"
)

type userDataPart struct {
	contentType string
	content     string
	filename    string
}

func buildFinalUserData(raw string, sshPublicKey string, sshUser string, disableInject bool) (string, error) {
	raw, err := loadUserData(raw)
	if err != nil {
		return "", err
	}

	rancherBootstrap := needsRancherBootstrap(raw)

	// When neither Rancher bootstrap nor SSH injection is requested, pass through raw user-data.
	if disableInject && !rancherBootstrap {
		return raw, nil
	}

	var parts []userDataPart
	if raw != "" {
		parts = append(parts, userDataPart{
			contentType: detectCloudInitContentType(raw),
			content:     raw,
		})
	}

	if rancherBootstrap {
		parts = append(parts, userDataPart{
			contentType: "text/x-shellscript-per-instance",
			content:     buildRancherBootstrapScript(),
			filename:    "10-makecloud-rancher-bootstrap.sh",
		})
	}

	if !disableInject {
		inject := buildSSHInjectCloudConfig(sshPublicKey, sshUser)
		if raw == "" && !rancherBootstrap {
			return inject, nil
		}
		parts = append(parts, userDataPart{
			contentType: "text/cloud-config",
			content:     inject,
		})
	}

	switch len(parts) {
	case 0:
		return "", nil
	case 1:
		return parts[0].content, nil
	default:
		return buildMultipartCloudInit(parts), nil
	}
}

func loadUserData(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	path := value
	if strings.HasPrefix(value, "@") {
		path = strings.TrimSpace(strings.TrimPrefix(value, "@"))
	}

	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read user-data file %q: %w", path, err)
		}
		return string(b), nil
	}

	return value, nil
}

func detectCloudInitContentType(content string) string {
	c := strings.TrimSpace(content)
	switch {
	case strings.HasPrefix(c, "#cloud-config"):
		return "text/cloud-config"
	case strings.HasPrefix(c, "#!"):
		return "text/x-shellscript"
	default:
		return "text/plain"
	}
}

func needsRancherBootstrap(raw string) bool {
	// Rancher machine provisioning injects a cloud-config that writes the install
	// script into /usr/local/custom_script/install.sh. On MakeCloud the floating
	// IP is NATed, and using it as the Kubernetes apiserver advertise address
	// breaks in-cluster access to the kubernetes service (10.43.0.1:443). We fix
	// both issues by:
	// 1) forcing advertise-address/node-ip to the VM private IP at boot,
	// 2) executing the install script once if it exists and system-agent is not installed yet.
	return strings.Contains(raw, "/usr/local/custom_script/install.sh")
}

func buildRancherBootstrapScript() string {
	return `#!/bin/sh
set -eu

# Prefer the primary private IPv4 of the VM. Floating IPs are typically NATed
# and not reachable from inside the guest, so they must not be used for the
# apiserver advertise address.
private_ip="$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}' || true)"
if [ -z "${private_ip}" ]; then
  private_ip="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
fi

if [ -n "${private_ip}" ]; then
  mkdir -p /etc/rancher/rke2/config.yaml.d
  printf "advertise-address: %s\nnode-ip:\n  - %s\n" "${private_ip}" "${private_ip}" >/etc/rancher/rke2/config.yaml.d/99-makecloud.yaml
fi

if [ -f /usr/local/custom_script/install.sh ] && [ ! -f /etc/systemd/system/rancher-system-agent.service ]; then
  sh /usr/local/custom_script/install.sh >/var/log/rancher-custom-install.log 2>&1 || true
fi
`
}

func buildSSHInjectCloudConfig(sshPublicKey string, sshUser string) string {
	sshPublicKey = strings.TrimSpace(sshPublicKey)
	sshUser = strings.TrimSpace(sshUser)
	if sshUser == "" {
		sshUser = "root"
	}

	if sshUser == "root" {
		escapedKey := strings.ReplaceAll(sshPublicKey, `'`, `'\''`)
		return fmt.Sprintf(`#cloud-config
runcmd:
  - [ sh, -c, "mkdir -p /root/.ssh && chmod 700 /root/.ssh && echo '%s' >> /root/.ssh/authorized_keys && chmod 600 /root/.ssh/authorized_keys" ]
`, escapedKey)
	}

	return fmt.Sprintf(`#cloud-config
users:
  - default
  - name: %s
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - %s
`, sshUser, sshPublicKey)
}

func buildMultipartCloudInit(parts []userDataPart) string {
	boundary := "===============makecloud=="

	var b strings.Builder
	b.WriteString("MIME-Version: 1.0\n")
	b.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\n", boundary))
	b.WriteString("\n")

	for i, p := range parts {
		if i == 0 {
			b.WriteString(fmt.Sprintf("--%s\n", boundary))
		} else {
			b.WriteString(fmt.Sprintf("\n--%s\n", boundary))
		}
		b.WriteString(fmt.Sprintf("Content-Type: %s\n", p.contentType))
		b.WriteString("Content-Transfer-Encoding: 7bit\n")
		if p.filename != "" {
			b.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\n", p.filename))
		} else {
			b.WriteString("Content-Disposition: attachment\n")
		}
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(p.content))
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\n--%s--\n", boundary))
	return b.String()
}
