package makecloud

import (
	"fmt"
	"os"
	"strings"
)

type userDataPart struct {
	contentType string
	content     string
}

func buildFinalUserData(raw string, sshPublicKey string, sshUser string, disableInject bool) (string, error) {
	raw, err := loadUserData(raw)
	if err != nil {
		return "", err
	}

	if disableInject {
		return raw, nil
	}

	inject := buildSSHInjectCloudConfig(sshPublicKey, sshUser)
	if raw == "" {
		return inject, nil
	}

	parts := []userDataPart{
		{
			contentType: detectCloudInitContentType(raw),
			content:     raw,
		},
		{
			contentType: "text/cloud-config",
			content:     inject,
		},
	}

	return buildMultipartCloudInit(parts), nil
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
		b.WriteString("Content-Disposition: attachment\n")
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(p.content))
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\n--%s--\n", boundary))
	return b.String()
}
