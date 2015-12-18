package pack
import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"encoding/json"
	"strings"
	"os/user"
	"log"
)

const (
	ProjectPackageFile = "package.json"

)
const (
	projectTempPrefix = "gopack"
)

// Project description
type Project struct {
	Descriptor   Descriptor
	PreInstall   []string
	PostInstall  []string
	PreRm        []string
	PreRemove    []string
	ReleaseNotes string
	WorkDir      string
}


// Read package description from file
func ReadPackage(file string) (Project, error) {
	prj := Project{}
	d, err := ioutil.ReadFile(file)
	if err != nil {
		return prj, err
	}
	err = json.Unmarshal(d, &(prj.Descriptor))
	if err != nil {
		return prj, err
	}
	prj.WorkDir = path.Dir(file)
	return prj, nil
}



// Prepare and build DEB package into result directory
func (pr *Project) Make(resultDir string) error {
	err := pr.Descriptor.FillDefault()
	if err != nil {
		return err
	}
	pr.Descriptor.FillTemplates()
	dir, err := ioutil.TempDir("", projectTempPrefix)
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	binFile := path.Join(dir, pr.Descriptor.TargetBinDir, pr.Descriptor.BinName)
	if err = os.MkdirAll(path.Dir(binFile), 0755); err != nil {
		return err
	}

	if err = pr.makeResources(dir); err != nil {
		return err
	}
	if pr.Descriptor.Service != nil {
		if err = pr.makeService(dir); err != nil {
			return err
		}
	}
	pr.makeReleaseNotes()
	if err = pr.makeControlFiles(dir); err != nil {
		return err
	}

	pr.PreInstall = append(pr.PreInstall, pr.Descriptor.PreInstall(), mustTemplate(getFileOrScript(pr.Descriptor.PreInst), pr.Descriptor))
	pr.PostInstall = append(pr.PostInstall, pr.Descriptor.PostInstall(), mustTemplate(getFileOrScript(pr.Descriptor.PostInst), pr.Descriptor))
	pr.PreRemove = append(pr.PreRemove, pr.Descriptor.PreRemove(), mustTemplate(getFileOrScript(pr.Descriptor.PreRm), pr.Descriptor))

	cmdGoGet := exec.Command("go", "get", pr.WorkDir)
	cmdGoGet.Stdout = os.Stdout
	cmdGoGet.Stderr = os.Stderr
	if err = cmdGoGet.Run(); err != nil {
		return err
	}

	for _, arch := range pr.Descriptor.Architectures {
		arch = strings.ToLower(arch)
		log.Println("Building for", arch)
		if err = os.Setenv("GOARCH", arch); err != nil {
			return err
		}

		cmdGoBuild := exec.Command("go", "build", "-o", binFile, pr.WorkDir)
		cmdGoBuild.Stdout = os.Stdout
		cmdGoBuild.Stderr = os.Stderr
		if err = cmdGoBuild.Run(); err != nil {
			return err
		}

		cmdDpkgBuild := exec.Command("dpkg", "-b", dir, path.Join(resultDir, pr.Descriptor.BinName + "-" + pr.Descriptor.Version + "_" + arch + ".deb"))
		cmdDpkgBuild.Stdout = os.Stdout
		cmdDpkgBuild.Stderr = os.Stderr
		if err = cmdDpkgBuild.Run(); err != nil {
			return err
		}

	}

	return nil
}

func (pr *Project) makeService(tmpDir string) error {
	sFile := path.Join(tmpDir, pr.Descriptor.TargetServiceDir, pr.Descriptor.ServiceFile())
	if err := os.MkdirAll(path.Dir(sFile), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(tmpDir, pr.Descriptor.TargetConfDir), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(sFile, []byte(pr.Descriptor.ServiceInit()), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(tmpDir, pr.Descriptor.TargetConfDir, ServiceConfigFile), []byte(pr.Descriptor.ServiceConfig()), 0755); err != nil {
		return err
	}
	return nil
}

func (pr *Project) makeControlFiles(tmpDir string) error {
	// DO debian files
	os.MkdirAll(path.Join(tmpDir, "DEBIAN"), 0755)
	if !isEmptyLines(pr.PreInstall) {
		if err := ioutil.WriteFile(path.Join(tmpDir, "DEBIAN", "preinst"), []byte(makeScript(pr.PreInstall)), 0755); err != nil {
			return err
		}
	}

	if !isEmptyLines(pr.PostInstall) {
		if err := ioutil.WriteFile(path.Join(tmpDir, "DEBIAN", "postinst"), []byte(makeScript(pr.PostInstall)), 0755); err != nil {
			return err
		}
	}

	if !isEmptyLines(pr.PreRemove) {
		if err := ioutil.WriteFile(path.Join(tmpDir, "DEBIAN", "prerm"), []byte(makeScript(pr.PreRemove)), 0755); err != nil {
			return err
		}
	}

	if pr.ReleaseNotes != "" {
		if err := ioutil.WriteFile(path.Join(tmpDir, "DEBIAN", "changelog"), []byte(pr.ReleaseNotes), 0700); err != nil {
			return err
		}
	}


	if err := ioutil.WriteFile(path.Join(tmpDir, "DEBIAN", "control"), []byte(pr.Descriptor.Control()), 0755); err != nil {
		return err
	}
	return nil
}

func (pr *Project) makeReleaseNotes() {
	cmd := exec.Command("git", "log", pr.WorkDir)
	cmd.Stderr = os.Stderr
	data, err := cmd.Output()
	if err != nil || !cmd.ProcessState.Success() {
		//silent error
		log.Println("failed git log", err)
		return
	}
	pr.ReleaseNotes = strings.TrimSpace(string(data))
}

func (pr *Project) makeResources(tmpDir string) error {
	resDir := path.Join(tmpDir, pr.Descriptor.TargetResourcesDir)
	if st, err := os.Stat(pr.Descriptor.Resources); err == nil && st.IsDir() {
		err = CopyDir(pr.Descriptor.Resources, resDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func newApp(nameGroup string) (Descriptor, error) {
	usr, err := user.Current()
	if err != nil {
		return Descriptor{}, err
	}
	parts := strings.SplitN(nameGroup, "-", 2)
	group := parts[0]
	name := parts[0]
	if len(parts) == 2 {
		name = parts[1]
	}
	return Descriptor{
		Name:name,
		Group:group,
		Author: usr.Name,
		Version:"1.0.0",
		Description: "Implementation of " + nameGroup    }, nil
}


// Initialize new application and save package description to specified directory.
// May parse package by following pattern: group-package_name
func SaveNewApp(dir, packet string) error {
	d, err := newApp(packet)
	if err != nil {
		return err
	}
	t, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(dir, ProjectPackageFile), []byte(t), 0700)
}

// Initialize new application with service and save package description to specified directory.
// May parse package by following pattern: group-package_name
func SaveNewService(dir, packet string) error {
	d, err := newApp(packet)
	if err != nil {
		return err
	}

	d.Service = &Service{
		AutoStart:true,
		Restart:true,
		RestartDelay:5,
		RunOpts:"",
		Env:map[string]string{},
	}
	t, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(dir, ProjectPackageFile), []byte(t), 0700)
}

func makeScript(lines []string) string {
	t := "#!/bin/bash"
	for _, p := range lines {
		t += "\n" + p
	}
	return t
}

func isEmptyLines(lines []string) bool {
	for _, l := range lines {
		if l != "" {
			return false
		}
	}
	return true
}

func getFileOrScript(line string) string {
	if line == "" {
		return ""
	}
	if content, err := ioutil.ReadFile(line); err == nil {
		return string(content)
	}
	return line
}