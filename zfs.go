package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

type ZfsDatasetInfo struct {
	name       string
	mountpoint string
	quota      *uint64
}

type ZfsClient struct {
	sshClient *ssh.Client
	sudo      bool
}

func (z *ZfsClient) CreateDataset(name string, properties map[string]string) error {
	args := []string{}
	args = append(args, "zfs", "create")
	for k, v := range properties {
		args = append(args, fmt.Sprintf("-o %s=%s", k, v))
	}
	args = append(args, name)
	_, err := z.runArgs(args)
	return err
}

func (z *ZfsClient) RenameDataset(prev, next string) error {
	args := []string{"zfs", "rename", prev, next}
	_, err := z.runArgs(args)
	if err != nil {
		log.Printf("Error renaming dataset %s to %s: %v", prev, next, err)
		return err
	}
	log.Printf("Renamed dataset %s to %s", prev, next)
	return nil
}

// find the first dataset whose properties match the given ones.
// returns the empty string if no dataset is found.
func (z *ZfsClient) FindDatasetByProperties(properties map[string]string) (string, error) {
	propertyNames := []string{}
	propertyNames = append(propertyNames, "name")
	for key, _ := range properties {
		propertyNames = append(propertyNames, key)
	}

	args := []string{"zfs", "list", "-H", "-t", "filesystem", "-o", strings.Join(propertyNames, ",")}
	output, err := z.runArgs(args)
	if err != nil {
		return "", err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		propertyValues := strings.Split(line, "\t")
		if len(propertyValues) != len(properties)+1 {
			log.Printf("zfs list returned invalid number of property values, expected %d but got %d", len(properties)+1, len(propertyValues))
			log.Printf("properties: %v", properties)
			log.Printf("line: %s", line)
			return "", fmt.Errorf("zfs list returned invalid number of property values, expected %d but got %d", len(properties)+1, len(propertyValues))
		}

		found := true
		datasetName := propertyValues[0]
		for i := 1; i < len(propertyNames); i += 1 {
			propertyName := propertyNames[i]
			propertyExpectedValue := properties[propertyName]
			propertyValue := propertyValues[i]
			if propertyValue != propertyExpectedValue {
				found = false
				break
			}
		}

		if found {
			return datasetName, nil
		}
	}

	return "", nil
}

func (z *ZfsClient) CreateDatasetIfNotExists(name string, properties map[string]string) error {
	exists, err := z.DatasetExists(name)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("Dataset already exists, skipping creation: %s", name)
		return nil
	} else {
		log.Printf("Dataset does not exist, creating: %s", name)
		return z.CreateDataset(name, properties)
	}
}

func (z *ZfsClient) ShareDataset(name string) error {
	args := []string{"zfs", "share", name}
	output, err := z.runArgs(args)
	soutput := string(output)
	if strings.Contains(soutput, "filesystem already shared") {
		return nil
	}
	return err
}

func (z *ZfsClient) ChmodDataset(name string, mode string) error {
	mountpoint, err := z.GetDatasetMountpoint(name)
	if err != nil {
		return err
	}
	args := []string{"chmod", mode, mountpoint}
	_, err = z.runArgs(args)
	return err
}

func (z *ZfsClient) SetDatasetQuota(name string, size int64) error {
	args := []string{"zfs", "set", fmt.Sprintf("quota=%d", size), name}
	_, err := z.runArgs(args)
	return err
}

func (z *ZfsClient) ListDatasets() ([]ZfsDatasetInfo, error) {
	return z.listDatasets("", 0)
}

func (z *ZfsClient) ListChildDatasets(parent string) ([]ZfsDatasetInfo, error) {
	info, err := z.listDatasets(parent, 1)
	if err != nil {
		return nil, err
	}
	return info[1:], nil
}

func (z *ZfsClient) DatasetExists(name string) (bool, error) {
	datasets, err := z.ListDatasets()
	if err != nil {
		return false, err
	}
	for _, dataset := range datasets {
		if dataset.name == name {
			return true, nil
		}
	}
	return false, nil
}

func (z *ZfsClient) GetDatasetMountpoint(name string) (string, error) {
	info, err := z.ListDatasets()
	if err != nil {
		return "", err
	}
	for _, dataset := range info {
		if dataset.name == name {
			return dataset.mountpoint, nil
		}
	}
	return "", fmt.Errorf("dataset not found: %s", name)
}

func (z *ZfsClient) GetProperty(name string, key string) (string, error) {
	args := []string{"zfs", "get", "-H", "-o", "value", key, name}
	output, err := z.runArgs(args)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(output)
	if value == "" {
		return "", fmt.Errorf("property not found: %s/%s", name, key)
	}
	return value, nil
}

func (z *ZfsClient) UpdateProperty(name, key, value string) error {
	args := []string{"zfs", "set", fmt.Sprintf("%s=%s", key, value), name}
	_, err := z.runArgs(args)
	return err
}

func (z *ZfsClient) UpdateProperties(name string, properties map[string]string) error {
	args := []string{"zfs", "set"}
	for k, v := range properties {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, name)
	_, err := z.runArgs(args)
	return err
}

func (z *ZfsClient) listDatasets(parent string, depth int) ([]ZfsDatasetInfo, error) {
	args := []string{"zfs", "list", "-H", "-o name,mountpoint,quota"}
	if depth > 0 {
		args = append(args, "-d", fmt.Sprintf("%d", depth))
	}
	if parent != "" {
		args = append(args, parent)
	}
	output, err := z.runArgs(args)
	if err != nil {
		return nil, err
	}
	info := []ZfsDatasetInfo{}
	output = strings.TrimSpace(output)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\t")
		quota, err := parseQuota(fields[2])
		if err != nil {
			log.Printf("Error parsing quota '%s': %v", fields[2], err)
			return nil, err
		}

		info = append(info, ZfsDatasetInfo{
			name:       fields[0],
			mountpoint: fields[1],
			quota:      quota,
		})
	}
	return info, nil
}

func (z *ZfsClient) commandFromArgs(args []string) string {
	command := strings.Join(args, " ")
	if z.sudo {
		command = "sudo " + command
	}
	return command
}

func (z *ZfsClient) runArgs(args []string) (string, error) {
	session, err := z.sshClient.NewSession()
	if err != nil {
		log.Printf("Error creating session: %v", err)
		return "", err
	}
	defer session.Close()

	command := z.commandFromArgs(args)
	log.Printf("Running command: %s", command)
	output, err := session.CombinedOutput(command)
	soutput := strings.TrimSpace(string(output))
	log.Printf("Command output: %s", soutput)
	return soutput, err
}

func parseQuota(quota string) (*uint64, error) {
	if quota == "none" {
		return nil, nil
	}

	multiplier := float64(1)
	if strings.HasSuffix(quota, "K") {
		quota = strings.TrimSuffix(quota, "K")
		multiplier = 1024
	}
	if strings.HasSuffix(quota, "M") {
		quota = strings.TrimSuffix(quota, "M")
		multiplier = 1024 * 1024
	}
	if strings.HasSuffix(quota, "G") {
		quota = strings.TrimSuffix(quota, "G")
		multiplier = 1024 * 1024 * 1024
	}
	if strings.HasSuffix(quota, "T") {
		quota = strings.TrimSuffix(quota, "T")
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	v, err := strconv.ParseFloat(quota, 64)
	if err != nil {
		return nil, err
	}
	v = v * multiplier

	total := uint64(v)
	return &total, nil
}
