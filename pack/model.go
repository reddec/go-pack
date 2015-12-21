package pack
import (
	"errors"
	"runtime"
	"os/user"
	"time"
	"strings"
	"strconv"
)
const ServiceConfigFile = "service.conf"
type Service struct {
	Env          map[string]string `json:"env,omitempty"`
	RunOpts      string`json:"opts,omitempty"`
	Restart      bool`json:"restart"`
	AutoStart    bool`json:"autostart"`
	RestartDelay int`json:"restartDelay"`
	TargetInit   string`json:"target,omitempty"`
}

// Full project description
type Descriptor struct {
	Group              string`json:"group"`
	Name               string`json:"name"`
	Version            string`json:"version"`
	Author             string`json:"author"`
	Description        string`json:"description"`
	Depends            []string `json:"depends,omitempty"`
	Architectures      []string `json:"arch,omitempty"`
	BinName            string`json:"bin,omitempty"`
	Service            *Service`json:"service,omitempty"`
	Resources          string`json:"resources,omitempty"`
	PreInst            string `json:"preinst,omitempty"`
	PostInst           string `json:"postinst,omitempty"`
	PreRm              string `json:"prerm,omitempty"`
	TargetResourcesDir string`json:"resourcesDir,omitempty"`
	TargetBinDir       string`json:"binDir,omitempty"`
	TargetConfDir      string`json:"confDir,omitempty"`
	TargetServiceDir   string `json:"serviceDir,omitempty"`
}


// Fill default fields in project description
func (d *Descriptor) FillDefault() error {
	if d.Resources == "" {
		d.Resources = "resources"
	}
	if d.Name == "" {
		return errors.New("Name must be specified")
	}
	if d.Group == "" {
		return errors.New("Group must be specified")
	}
	if d.Author == "" {
		usr, err := user.Current()
		if err != nil {
			return err
		}
		d.Author = usr.Name
	}
	if d.Version == "" {
		d.Version = "0.0.0"
	}
	if d.Description == "" {
		d.Description = "{{.Group}} {{.Name}} built at " + time.Now().Format(time.RFC3339)
	}
	if len(d.Architectures) == 0 {
		d.Architectures = append(d.Architectures, runtime.GOARCH)
	}
	if d.BinName == "" {
		d.BinName = "{{.Group}}-{{.Name}}"
	}
	if d.Service != nil {
		d.Service.FillDefault()
	}
	if d.TargetResourcesDir == "" {
		d.TargetResourcesDir = "/usr/local/share/{{.Group}}/{{.Name}}"
	}
	if d.TargetBinDir == "" {
		d.TargetBinDir = "/usr/local/bin"
	}
	if d.TargetConfDir == "" {
		d.TargetConfDir = "/etc/{{.Group}}/{{.Name}}"
	}
	if d.TargetServiceDir == "" {
		d.TargetServiceDir = "/etc/init"
	}
	return nil
}

// Fill default values for services
func (s *Service) FillDefault() {
	if s.TargetInit == "" {
		s.TargetInit = "upstart"
	}else if s.TargetInit != "upstart" {
		panic("Only 'upstart' target init system is allowed for services")
	}
}


// Render all string fields as templates.
// May panic if some templates are wrong
func (d *Descriptor) FillTemplates() {
	d.mustTemplate(&(d.Name))
	d.mustTemplate(&(d.Resources))
	d.mustTemplate(&(d.Group))
	d.mustTemplate(&(d.Version))
	d.mustTemplate(&(d.BinName))
	d.mustTemplate(&(d.Description))
	d.mustTemplate(&(d.Author))
	d.mustTemplate(&(d.TargetServiceDir))
	d.mustTemplate(&(d.TargetResourcesDir))
	d.mustTemplate(&(d.TargetBinDir))
	d.mustTemplate(&(d.TargetConfDir))
	if d.Service != nil {
		d.mustTemplate(&(d.Service.TargetInit))
		for k, v := range d.Service.Env {
			tmp := v
			d.mustTemplate(&tmp)
			d.Service.Env[k] = tmp
		}
		d.Service.RunOpts = mustTemplate(d.Service.RunOpts, *d)
	}


}

// Get DEBIAN architecture name
func normalizeArch(arch string) string {
	if _, err := strconv.ParseUint(arch, 10, 64); err != nil {
		return arch
	}
	return "i" + arch
}

// Content of Control file in DEB package
func (d *Descriptor) Control() string {
	t := `Package: {{.Group}}-{{.Name}}
Version: {{.Version}}
Architecture: ` + normalizeArch(runtime.GOARCH) + `
Maintainer: {{.Author}}
`
	if len(d.Depends) != 0 {
		t += "Depends: " + strings.Join(d.Depends, ",") + "\n"
	}
	t += `Description: {{.Description}}
`
	d.mustTemplate(&t)
	return t
}

// Content of pre-install script
func (d *Descriptor) PreInstall() string {
	t := ""
	if d.Service != nil {
		t += mustTemplate("service {{.BinName}} stop || echo 'No installed {{.Group}} {{.Name}} instance running'\n", *d)
		// Backup

		t += mustTemplate(`# backup old configuration
BACKUP_DIR="/tmp/backups/{{.Group}}/{{.Name}}"
if [ -d "$BACKUP_DIR" ]; then
    rm -rf $BACKUP_DIR
    mkdir $BACKUP_DIR
    cp -r "{{.TargetConfDir}}"/* "$BACKUP_DIR"/
fi
`, *d)
	}
	return t
}

// Content of post-install script
func (d *Descriptor) PostInstall() string {
	t := ""
	if d.Service != nil {
		if d.Service.AutoStart {
			t += mustTemplate("service {{.BinName}} start || echo 'No {{.Group}} {{.Name}} instance'\n", *d)
		}
		t += mustTemplate(`# restore old configuration
BACKUP_DIR="/tmp/backups/{{.Group}}/{{.Name}}"
if [ -d "$BACKUP_DIR" ]; then
    cp -r "$BACKUP_DIR"/* "{{.TargetConfDir}}"/ && rm -rf "$BACKUP_DIR"
fi
`, *d)
	}
	return t
}

// Content of pre-remove script
func (d *Descriptor) PreRemove() string {
	return d.PreInstall()
}

// Content of service .conf file for init system
func (d *Descriptor) ServiceInit() string {
	t := `# {{.Group}} {{.Name}}

description         "{{.Description}}"

start on runlevel [2345]
stop on runlevel [!2345]
{{if .Service.Restart}}
respawn
respawn limit 99999999 {{.Service.RestartDelay}}
{{end}}

script
  . {{.TargetConfDir}}/` + ServiceConfigFile + `
  exec {{.TargetBinDir}}/{{.BinName}} {{.Service.RunOpts}} | logger -t '{{.Group}}-{{.Name}}' 2>&1
end script
`
	return mustTemplate(t, *d)
}

// Service filename in init system
func (d *Descriptor) ServiceFile() string {
	t := `{{.Group}}-{{.Name}}.conf`
	return mustTemplate(t, *d)
}

// Content of configuration file for service
func (d *Descriptor) ServiceConfig() string {
	t := `#!/bin/bash
{{ range $key, $value := .Service.Env}}
export {{$key}}="{{$value}}"{{end}}
`
	return mustTemplate(t, *d)
}
