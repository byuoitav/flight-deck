package helpers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/byuoitav/authmiddleware/bearertoken"
	"github.com/byuoitav/common/db"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/structs"
	mapset "github.com/deckarep/golang-set"
)

var (
	filePath string
)

func init() {
	ex, err := os.Executable()
	if err != nil {
		log.L.Fatalf("Failed to get location of executable: %v", err)
	}

	filePath = filepath.Dir(ex)
}

// retrieveEnvironmentVariables gets the environment variables for each Pi as a file to SCP over
func retrieveEnvironmentVariables(deviceType, designation string) (map[string]string, error) {
	myMap := make(map[string]string)
	deviceInfo, err := db.GetDB().GetDeviceDeploymentInfo(deviceType)
	if err != nil {
		return myMap, err
	}
	desigDevice := deviceInfo.Designations[designation]
	for _, service := range desigDevice.DockerServices {
		resp, err := MakeEnvironmentRequest(service, designation)
		if err != nil {
			return myMap, err
		}

		for k, v := range resp {
			myMap[k] = v
		}
	}
	for k, v := range desigDevice.EnvironmentVariables {
		myMap[k] = v
	}
	return myMap, nil
}

func addMap(a, b map[string]interface{}) error {
	var s string
	set := mapset.NewSet(s)
	for k1 := range a {
		for k2, v2 := range b {
			if k1 == k2 {
				a[k1] = v2
				set.Add(k1)
			}
		}
	}
	for k, v := range b {
		if !set.Contains(k) {
			a[k] = v
		}
	}
	return nil
}

func substituteEnvironment(byter *bytes.Buffer, arrayV []interface{}, service string, tabCount int, envMap map[string]string) error {
	byter.WriteString("\n")
	for _, listItem := range arrayV {
		for i := 0; i < tabCount; i++ {
			byter.WriteString("   ")
		}
		strVersion := listItem.(string)
		values := strings.Split(strVersion, "=$")
		if len(values) < 2 {
			return fmt.Errorf("Values too short (did you forget a $ in the environment values?)")
		}
		str := fmt.Sprintf("  - %s=%s\n", values[0], envMap[values[1]])
		byter.WriteString(str)
	}
	return nil
}

func writeServiceMap(byter *bytes.Buffer, myMap map[string]interface{}, tabCount int, service string, envMap map[string]string) error {
	for k, v := range myMap {
		for i := 0; i < tabCount; i++ {
			byter.WriteString("  ")
		}
		s := fmt.Sprintf("%s:", k)
		byter.WriteString(s)
		if _, ok := v.(string); ok {
			str := fmt.Sprintf(" %s\n", v)
			byter.WriteString(str)
		}
		if _, ok := v.([]interface{}); ok {
			//If we have environment variables, do the appropriate substitution
			arrayV := v.([]interface{})
			if k == "environment" {
				substituteEnvironment(byter, arrayV, service, tabCount, envMap)
			} else {
				byter.WriteString("\n")

				for _, listItem := range arrayV {
					if _, ok = listItem.(string); ok {
						for i := 0; i < tabCount; i++ {
							byter.WriteString("  ")
						}
						strVersion := listItem.(string)
						str := fmt.Sprintf("  - %s\n", strVersion)
						byter.WriteString(str)

					} else {
						mapped := listItem.(map[string]interface{})
						first := true
						for mk, mv := range mapped {
							for i := 0; i < tabCount; i++ {
								byter.WriteString("  ")
							}
							if first {
								byter.WriteString("  - ")
							} else {
								byter.WriteString("    ")
							}
							first = false
							byter.WriteString(fmt.Sprintf("%s: %s\n", mk, mv))
						}
					}
				}
			}

		}
		if _, ok := v.(map[string]interface{}); ok {
			newMap := v.(map[string]interface{})
			byter.WriteString("\n")
			err := writeServiceMap(byter, newMap, (tabCount + 1), service, envMap)
			if err != nil {
				log.L.Warnf("Couldn't write to service map %v", err)
				return err
			}
		}
	}
	return nil
}

func writeMap(byter *bytes.Buffer, myMap map[string]interface{}, tabCount int, designation string, deviceType string) error {
	for k, v := range myMap {
		for i := 0; i < tabCount; i++ {
			byter.WriteString("  ")
		}
		s := fmt.Sprintf("%s:", k)
		byter.WriteString(s)
		_, ok := v.(string)
		if ok {
			str := fmt.Sprintf(" %s\n", v)
			byter.WriteString(str)
		}
		_, ok = v.([]interface{})
		if ok {
			arrayV := v.([]interface{})
			byter.WriteString("\n")

			for _, listItem := range arrayV {
				for i := 0; i < tabCount; i++ {
					byter.WriteString("   ")
				}
				strVersion := listItem.(string)
				str := fmt.Sprintf("  - %s\n", strVersion)
				byter.WriteString(str)
			}
		}
		_, ok = v.(map[string]interface{})
		if ok {
			newMap := v.(map[string]interface{})
			byter.WriteString("\n")

			resp, err := MakeEnvironmentRequest(k, designation)
			if err != nil {
				log.L.Warnf("Couldn't make the enviroment request: %v", err)
				return err
			}
			deviceInfo, err := db.GetDB().GetDeviceDeploymentInfo(deviceType)
			if err != nil {
				log.L.Warnf("Couldn't get the device deployment info: %v", err)
				return err
			}
			desigDevice := deviceInfo.Designations[designation]
			for k, v := range desigDevice.EnvironmentVariables {
				resp[k] = v
			}
			err = writeServiceMap(byter, newMap, (tabCount + 1), k, resp)
			if err != nil {
				log.L.Warnf("Error writing service map: %v", err)
			}
		}
	}
	return nil
}

//RetrieveDockerCompose .
func RetrieveDockerCompose(deviceType, designation string) ([]byte, error) {
	var b []byte
	var byter bytes.Buffer
	deviceInfo, err := db.GetDB().GetDeviceDeploymentInfo(deviceType)
	if err != nil {
		log.L.Warnf("Couldn't get the %s %s out of the database", designation, deviceType)
		return b, err
	}
	desigDevice := deviceInfo.Designations[designation]
	m := make(map[string]interface{})
	for _, service := range desigDevice.DockerServices {
		resp, err := MakeDockerRequest(service, designation)
		if err != nil {
			log.L.Warnf("Couldn't get the docker info for %s:%s", service, designation)
			return b, err
		}
		tempM := make(map[string]interface{})
		tempM[service] = resp
		addMap(m, tempM)
	}
	addMap(m, desigDevice.DockerInfo)
	byter.WriteString("version: '3.2'\n")
	byter.WriteString("services:\n")
	writeMap(&byter, m, 1, designation, deviceType)

	return byter.Bytes(), nil
}

// MakeEnvironmentRequest .
func MakeEnvironmentRequest(serviceID, designation string) (map[string]string, error) {
	resp, err := db.GetDB().GetDeploymentInfo(serviceID)
	if err != nil {
		log.L.Warnf("Couldn't get deployment info for %v %v: %v", designation, serviceID, err)
		return nil, err
	}
	if _, ok := resp.CampusConfig[designation]; !ok {
		return nil, fmt.Errorf("Designation doesn't exist for %v %v", designation, serviceID)
	}
	toReturn := resp.CampusConfig[designation].EnvironmentVariables
	if toReturn == nil {
		return nil, fmt.Errorf("Environment Variables empty for %v %v", designation, serviceID)
	}
	return toReturn, nil
}

// MakeDockerRequest .
func MakeDockerRequest(serviceID, designation string) (map[string]interface{}, error) {
	resp, err := db.GetDB().GetDeploymentInfo(serviceID)
	if err != nil {
		log.L.Warnf("Couldn't get deployment info for %v %v: %v", designation, serviceID, err)
		return nil, err
	}
	if _, ok := resp.CampusConfig[designation]; !ok {
		return nil, fmt.Errorf("Designation doesn't exist for %v %v", designation, serviceID)
	}
	toReturn := resp.CampusConfig[designation].DockerInfo
	if toReturn == nil {
		return nil, fmt.Errorf("Dockerinfo empty for %v %v", designation, serviceID)
	}
	return toReturn, err
}

// SetToken .
func SetToken(request *http.Request) error {

	//	log.Printf("[helpers] setting bearer token...")

	token, err := bearertoken.GetToken()
	if err != nil {
		msg := fmt.Sprintf("cannot get bearer token: %s", err.Error())
		//		log.Printf("%s", color.HiRedString("[helpers] %s", msg))
		return errors.New(msg)
	}

	request.Header.Set("Authorization", "Bearer "+token.Token)

	return nil
}

// GetServiceFromCouch .
func GetServiceFromCouch(service, designation, deviceType, deviceID string) ([]file, bool, error) {
	files := []file{}
	serviceFileExists := false
	log.L.Infof("Getting files in Couch from %s/%s", designation, service)

	objects, err := GetCouchServiceFiles(service, designation, deviceType, deviceID)
	if err != nil {
		return nil, serviceFileExists, fmt.Errorf("unable to download service %s (designation: %s) from couch: %s", service, designation, err)
	}

	for name, bytes := range objects {
		file := file{
			Path:  fmt.Sprintf("/byu/%s/%s", service, name),
			Bytes: bytes,
		}
		log.L.Debugf("Service Name: %s\n", name)
		if name == service {
			file.Permissions = 0100
		} else if name == fmt.Sprintf("%s.service", service) {
			serviceFileExists = true
			file.Permissions = 0644
		} else {
			file.Permissions = 0644
		}

		log.L.Debugf("added file %v, permissions %v", file.Path, file.Permissions)
		files = append(files, file)
	}

	log.L.Infof("Successfully got %v files.", len(files))
	return files, serviceFileExists, nil
}

func serviceTemplateEnvSwap(value string, envMap map[string]string, deviceID string) string {
	if value == "$SYSTEM_ID" {
		return deviceID
	}
	if strings.Contains(value, "$") {
		cleanValue := strings.Split(value, "$")
		return envMap[cleanValue[1]]
	}
	return value

}

// I've basically given up on giving good names to these functions

func writeServiceTemplate(byter *bytes.Buffer, serviceConfig structs.ServiceConfig, deviceType, designation, deviceID string) error {
	envMap, err := retrieveEnvironmentVariables(deviceType, designation)
	if err != nil {
		return err
	}
	for k, v := range serviceConfig.Data {
		byter.WriteString(fmt.Sprintf("[%s]\n", k))
		for key, value := range v {
			if isEnvironment := strings.Split(key, "="); len(isEnvironment) == 2 {
				byter.WriteString(fmt.Sprintf("%s=%s\n", key, serviceTemplateEnvSwap(value, envMap, deviceID)))
			} else {
				byter.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			}
		}
		byter.WriteString("\n")
	}
	return nil
}

type couchService struct {
	service     string
	designation string
}

var (
	services   map[couchService][]byte
	servicesMu sync.Mutex
)

func init() {
	services = make(map[couchService][]byte)
}

// GetCouchServiceFiles .
func GetCouchServiceFiles(service, designation, deviceType, deviceID string) (map[string][]byte, error) {

	couchService := couchService{
		service:     service,
		designation: designation,
	}

	objects := make(map[string][]byte)

	//Handle Service Template
	toFill, err := db.GetDB().GetServiceInfo(service)
	if err != nil {
		log.L.Warnf("Couldn't get the service data from Couch: %v", err)
		return objects, err
	}
	serviceConfig := toFill.Designations[designation]
	var byter bytes.Buffer
	writeServiceTemplate(&byter, serviceConfig, deviceType, designation, deviceID)
	objects[fmt.Sprintf("%v.service", service)] = byter.Bytes()

	servicesMu.Lock()
	tarball, ok := services[couchService]
	if !ok {
		tarball, err = db.GetDB().GetServiceZip(service, designation)
		if err != nil {
			log.L.Warnf("Couldn't get the tarball from couch: %v", err)
			servicesMu.Unlock()
			return objects, err
		}

		services[couchService] = tarball
	}
	servicesMu.Unlock()

	//Handle tar.gz
	gzf, err := gzip.NewReader(bytes.NewReader(tarball))
	if err != nil {
		log.L.Warnf("Coudn't make gzip reader for %v: %v", service, err)
		return nil, err
	}

	tr := tar.NewReader(gzf)

	// Read all the files from archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.L.Warnf("Couldn't read file: %v", err)
			break
		}

		log.L.Debugf("Reading file: %v", header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			buf := &bytes.Buffer{}

			n, err := io.Copy(buf, tr)
			switch {
			case err != nil:
				log.L.Warnf("unable to read the bytes of %v: %v\n", header.Name)
			case n != header.Size:
				log.L.Warnf("failed to read all bytes of %v: read %v, expected %v", header.Name, n, header.Size)
			}

			objects[header.Name] = buf.Bytes()
		}

	}

	return objects, nil
}
