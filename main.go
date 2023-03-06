package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	logsDir := getDirectory()
	fmt.Println(extractSteamData(logsDir))
}

func getDirectory() string {
	s := bufio.NewScanner(os.Stdin)
	fmt.Println("Директория с логами: ")
	s.Scan()
	return s.Text()
}

func extractSteamData(logsFolder string) error {
	logsFolders, err := os.ReadDir(logsFolder)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll("./Results", os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("Неудалось создать папку для сохранения результатов: %v\n", err)
	}

	folderIndex := 1
	for _, logFoldlogsFolder := range logsFolders {
		logFolderPath := filepath.Join(logsFolder, logFoldlogsFolder.Name())
		passwordFile := filepath.Join(logFolderPath, "Passwords.txt")
		steamFolderPath := filepath.Join(logFolderPath, "Steam")

		if _, err := os.Stat(steamFolderPath); err != nil {
			log.Printf("В логе %s нету папки Steam", logFoldlogsFolder.Name())
			continue
		}

		if _, err := os.Stat(passwordFile); err != nil {
			log.Printf("В логе %s нету Passwords.txt", logFoldlogsFolder.Name())
			continue
		}

		steamFolders, err := os.ReadDir(steamFolderPath)
		if err != nil {
			log.Println(logFolderPath, err)
			continue
		}

		for _, steamFolder := range steamFolders {
			if strings.Contains(steamFolder.Name(), "loginusers.vdf") {
				arrSteamAccounts, err := getSteamAccounts(logFolderPath)
				if err != nil {
					return fmt.Errorf("Произошла ошибка при получение данных аккаунтов: %v", err)
				}

				if len(arrSteamAccounts) == 0 {
					continue
				}

				folderIndexString := strconv.Itoa(folderIndex)
				dirName := filepath.Join("./Results", folderIndexString)
				err = os.Mkdir(dirName, os.FileMode(0755))
				if err != nil {
					return fmt.Errorf("Неудалось создать подпапку для сохранения результата: %v\n", err)
				}

				loginusersFile, err := os.Open(filepath.Join(steamFolderPath, steamFolder.Name()))
				if err != nil {
					log.Println(steamFolderPath, err)
				}
				defer loginusersFile.Close()

				loginusersResultPath := filepath.Join(dirName, "loginusers.vdf")
				loginusersResultFile, err := os.Create(loginusersResultPath)
				if err != nil {
					return fmt.Errorf("Неудалось создать файл для записи loginusers.vdf: %v\n", err)
				}
				defer loginusersResultFile.Close()

				if _, err := io.Copy(loginusersResultFile, loginusersFile); err != nil {
					return fmt.Errorf("Неудалось скопировать содержимое файла loginusers.vdf: %v\n", err)
				}

				configFilePath := filepath.Join(steamFolderPath, "config.vdf")
				configFile, err := os.Open(configFilePath)
				if err != nil {
					log.Println(steamFolderPath, err)
				}
				defer configFile.Close()

				configResultPath := filepath.Join(dirName, "config.vdf")
				configResultFile, err := os.Create(configResultPath)
				if err != nil {
					return fmt.Errorf("Неудалось создать файл для записи config.vdf: %v\n", err)
				}
				defer configResultFile.Close()

				if _, err := io.Copy(configResultFile, configFile); err != nil {
					return fmt.Errorf("Неудалось скопировать содержимое файла config.vdf: %v\n", err)
				}

				err = filepath.Walk(steamFolderPath, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if !info.IsDir() && strings.HasPrefix(info.Name(), "ssfn") {
						ssfnResultPath := filepath.Join(dirName, info.Name())
						ssfnResultFile, err := os.Create(ssfnResultPath)
						if err != nil {
							return fmt.Errorf("Неудалось создать файл для записи %s: %v\n", info.Name(), err)
						}
						defer ssfnResultFile.Close()

						ssfnFile, err := os.Open(path)
						if err != nil {
							log.Println(steamFolderPath, err)
						}
						defer ssfnFile.Close()
						if _, err := io.Copy(ssfnResultFile, ssfnFile); err != nil {
							return fmt.Errorf("Неудалось скопировать содержимое файла %s: %v\n", info.Name(), err)
						}
					}
					return nil
				})
				if err != nil {
					log.Println(steamFolderPath, err)
				}

				steamAccountsPath := dirName + "/LogPass.txt"
				steamAccountsFile, err := os.Create(steamAccountsPath)
				if err != nil {
					return fmt.Errorf("Произошла ошибка при создании файла для записи данных аккаунта: %v", err)
				}
				defer steamAccountsFile.Close()

				steamAccountsFile.WriteString(strings.Join(arrSteamAccounts, "\n"))

				logPath := dirName + "/Path.txt"
				logPathFile, err := os.Create(logPath)
				if err != nil {
					return fmt.Errorf("Произошла ошибка при создании файла для записи путей к логам: %v", err)
				}
				defer logPathFile.Close()

				logPathFile.WriteString(logFolderPath)

				folderIndex += 1
			}
		}
	}

	return nil
}

func getSteamAccounts(dirPath string) ([]string, error) {
	file, err := os.Open(filepath.Join(dirPath, "Passwords.txt"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var accounts []string
	var currentAccount string
	var inSteamBlock bool

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "URL: ") && strings.Contains(line, "steam") {
			currentAccount = ""
			inSteamBlock = true
		} else if inSteamBlock && strings.HasPrefix(line, "Username: ") {
			username := strings.TrimPrefix(line, "Username: ")
			currentAccount = username + ":"
		} else if inSteamBlock && strings.HasPrefix(line, "Password: ") {
			password := strings.TrimPrefix(line, "Password: ")
			currentAccount += password
			if currentAccount != ":" {
				accounts = append(accounts, currentAccount)
			}
			inSteamBlock = false
		} else {
			inSteamBlock = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}
