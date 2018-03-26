package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type Dotfiles struct {
	Applications []string
	Configs      []string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func loadDotfiles() (dotfiles Dotfiles, err error) {
	data, err := ioutil.ReadFile("dotfiles.yaml")
	if err != nil {
		return
	}

	err = yaml.Unmarshal([]byte(data), &dotfiles)
	return
}

func getHomeDir() (homeDir string, err error) {
	usr, err := user.Current()
	if err != nil {
		return
	}

	homeDir = usr.HomeDir
	return
}

func main() {
	dotfiles, err := loadDotfiles()
	check(err)

	homeDir, err := getHomeDir()
	check(err)

	fmt.Printf("Installing applications:\n")
	for idx := range dotfiles.Applications {
		packageName := dotfiles.Applications[idx]
		fmt.Printf("Installing %s...\n", packageName)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("sudo eopkg install %s", packageName))
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		check(err)
	}

	fmt.Printf("Symlinking config files:\n")
	for idx := range dotfiles.Configs {
		configDestination := strings.Replace(dotfiles.Configs[idx], "~", homeDir, 1)
		configSource, fileInfo, err := findConfig(configDestination)
		check(err)

		if configSource != "" {
			// Make sure that directory of the config file exists
			err = os.MkdirAll(filepath.Dir(configDestination), os.ModePerm)
			check(err)

			configType := "file"
			if fileInfo.IsDir() {
				configType = "dir"
				os.Remove(configDestination)
				configDestination = filepath.Dir(configDestination)
			}

			fmt.Printf("Symlinking config %s: %s -> %s\n", configType, configSource, configDestination)
			cmd := exec.Command("sh", "-c", fmt.Sprintf("ln -sfn %s %s", configSource, configDestination))
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			check(err)
		}
	}
}

func findConfig(configFilePath string) (result string, fileInfo os.FileInfo, err error) {
	rootDir, err := os.Getwd()
	check(err)

	var searchTerm string

	configFileName := filepath.Base(configFilePath)
	if configFileName == "config" {
		searchTerm = filepath.Base(filepath.Dir(configFilePath)) + "/" + configFileName
	} else {
		searchTerm = configFileName
	}

	err = filepath.Walk(rootDir, func(path string, f os.FileInfo, _ error) error {
		if f.IsDir() && f.Name() == ".git" {
			return filepath.SkipDir
		}

		match, err := regexp.MatchString(searchTerm, path)
		if err == nil && match {
			result = path
			fileInfo = f
			return io.EOF
		}

		return nil
	})

	if err == io.EOF {
		err = nil
	}

	return
}
