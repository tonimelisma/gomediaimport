package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

type volumesCmd struct {
	List *volumesListCmd `arg:"subcommand:list" help:"List currently mounted removable volumes"`
	Add  *volumesAddCmd  `arg:"subcommand:add" help:"Save a mounted removable volume label"`
}

type volumesListCmd struct{}

type volumesAddCmd struct {
	Selector string `arg:"positional,required" help:"Volume label or ID from volumes list"`
	DestDir  string `arg:"--dest" help:"Destination directory for this volume label"`
}

type removableVolumeConfig struct {
	DestDir string `yaml:"destination_directory,omitempty"`
}

type mountedRemovableVolume struct {
	Label     string
	MountPath string
}

type removableVolumeImport struct {
	Label     string
	SourceDir string
	DestDir   string
}

var mountedRemovableVolumes = listMountedRemovableVolumes
var importMediaRunner = importMedia

func runVolumesCommand(cmd *volumesCmd, cfg config) error {
	switch {
	case cmd.List != nil:
		return listRemovableVolumes(cfg, os.Stdout)
	case cmd.Add != nil:
		return addRemovableVolume(cfg, cmd.Add)
	default:
		return fmt.Errorf("volumes subcommand is required")
	}
}

func listRemovableVolumes(cfg config, out io.Writer) error {
	volumes, err := sortedMountedRemovableVolumes()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tLABEL\tSOURCE\tSAVED\tDESTINATION")
	for i, volume := range volumes {
		saved := "no"
		dest := "-"
		if entry, ok := cfg.RemovableVolumes[volume.Label]; ok && volume.Label != "" {
			saved = "yes"
			dest = entry.DestDir
			if dest == "" {
				dest = cfg.DestDir
			}
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", i+1, volume.Label, volume.MountPath, saved, dest)
	}
	return w.Flush()
}

func addRemovableVolume(cfg config, cmd *volumesAddCmd) error {
	volumes, err := sortedMountedRemovableVolumes()
	if err != nil {
		return err
	}

	label, err := resolveVolumeSelector(cmd.Selector, volumes)
	if err != nil {
		return err
	}
	if label == "" {
		return fmt.Errorf("selected removable volume does not have a label")
	}

	if cmd.DestDir != "" {
		destParent := filepath.Dir(cmd.DestDir)
		if _, err := os.Stat(destParent); os.IsNotExist(err) {
			return fmt.Errorf("destination parent directory does not exist: %s", destParent)
		}
	}

	if err := writeRemovableVolumeConfig(cfg.ConfigFile, label, cmd.DestDir, cmd.DestDir != ""); err != nil {
		return err
	}

	if cmd.DestDir == "" {
		fmt.Printf("Saved removable volume label %q using default destination.\n", label)
	} else {
		fmt.Printf("Saved removable volume label %q with destination %s.\n", label, cmd.DestDir)
	}
	return nil
}

func resolveVolumeSelector(selector string, volumes []mountedRemovableVolume) (string, error) {
	if id, err := strconv.Atoi(selector); err == nil {
		if id < 1 || id > len(volumes) {
			return "", fmt.Errorf("volume ID %d is not listed", id)
		}
		return volumes[id-1].Label, nil
	}

	for _, volume := range volumes {
		if volume.Label == selector {
			return selector, nil
		}
	}
	return "", fmt.Errorf("removable volume label %q is not currently mounted", selector)
}

func sortedMountedRemovableVolumes() ([]mountedRemovableVolume, error) {
	volumes, err := mountedRemovableVolumes()
	if err != nil {
		return nil, err
	}
	sort.Slice(volumes, func(i, j int) bool {
		if volumes[i].Label != volumes[j].Label {
			return volumes[i].Label < volumes[j].Label
		}
		return volumes[i].MountPath < volumes[j].MountPath
	})
	return volumes, nil
}

func importConfiguredRemovableVolumes(cfg config) error {
	imports, err := plannedRemovableVolumeImports(cfg)
	if err != nil {
		return err
	}

	if len(imports) == 0 {
		if !cfg.Quiet {
			fmt.Println("No configured removable volumes are currently mounted.")
		}
		return nil
	}

	var importErrors []error
	for _, volumeImport := range imports {
		importCfg := cfg
		importCfg.SourceDir = volumeImport.SourceDir
		importCfg.DestDir = volumeImport.DestDir

		if !cfg.Quiet {
			fmt.Printf("Importing removable volume %q from %s to %s\n", volumeImport.Label, volumeImport.SourceDir, volumeImport.DestDir)
		}

		if err := validateConfig(&importCfg); err != nil {
			importErrors = append(importErrors, fmt.Errorf("invalid configuration for volume %q at %s: %w", volumeImport.Label, volumeImport.SourceDir, err))
			continue
		}
		if err := importMediaRunner(importCfg); err != nil {
			importErrors = append(importErrors, fmt.Errorf("importing volume %q from %s: %w", volumeImport.Label, volumeImport.SourceDir, err))
		}
	}

	if len(importErrors) > 0 {
		return errors.Join(importErrors...)
	}
	return nil
}

func plannedRemovableVolumeImports(cfg config) ([]removableVolumeImport, error) {
	volumes, err := sortedMountedRemovableVolumes()
	if err != nil {
		return nil, err
	}

	labels := make([]string, 0, len(cfg.RemovableVolumes))
	for label := range cfg.RemovableVolumes {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	var imports []removableVolumeImport
	for _, label := range labels {
		entry := cfg.RemovableVolumes[label]
		destDir := entry.DestDir
		if destDir == "" {
			destDir = cfg.DestDir
		}
		for _, volume := range volumes {
			if volume.Label == label {
				imports = append(imports, removableVolumeImport{
					Label:     label,
					SourceDir: volume.MountPath,
					DestDir:   destDir,
				})
			}
		}
	}

	return imports, nil
}

func writeRemovableVolumeConfig(configPath, label, destDir string, setDest bool) error {
	doc, err := readConfigYAMLNode(configPath)
	if err != nil {
		return err
	}

	root := ensureDocumentMapping(doc)
	volumesNode := ensureMappingValue(root, "removable_volumes")
	entryNode, existed := mappingValue(volumesNode, label)
	if !existed {
		entryNode = &yaml.Node{Kind: yaml.MappingNode, Style: yaml.FlowStyle}
		appendMappingPair(volumesNode, label, entryNode)
	}

	if setDest {
		entryNode.Style = 0
		ensureScalarValue(entryNode, "destination_directory", destDir)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc); err != nil {
		_ = encoder.Close()
		return fmt.Errorf("failed to encode config file: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to encode config file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func readConfigYAMLNode(configPath string) (*yaml.Node, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return &doc, nil
}

func ensureDocumentMapping(doc *yaml.Node) *yaml.Node {
	if doc.Kind == 0 {
		doc.Kind = yaml.DocumentNode
	}
	if doc.Kind != yaml.DocumentNode {
		original := *doc
		*doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{&original}}
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	return doc.Content[0]
}

func ensureMappingValue(parent *yaml.Node, key string) *yaml.Node {
	value, ok := mappingValue(parent, key)
	if ok && value.Kind == yaml.MappingNode {
		return value
	}
	newValue := &yaml.Node{Kind: yaml.MappingNode}
	if ok {
		setMappingValue(parent, key, newValue)
		return newValue
	}
	appendMappingPair(parent, key, newValue)
	return newValue
}

func ensureScalarValue(parent *yaml.Node, key, value string) {
	scalar := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	if _, ok := mappingValue(parent, key); ok {
		setMappingValue(parent, key, scalar)
		return
	}
	appendMappingPair(parent, key, scalar)
}

func mappingValue(parent *yaml.Node, key string) (*yaml.Node, bool) {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			return parent.Content[i+1], true
		}
	}
	return nil, false
}

func setMappingValue(parent *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			parent.Content[i+1] = value
			return
		}
	}
	appendMappingPair(parent, key, value)
}

func appendMappingPair(parent *yaml.Node, key string, value *yaml.Node) {
	parent.Content = append(parent.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
}
