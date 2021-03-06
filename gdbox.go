package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var kMetadata = make(map[string]Metadata)
var kConfig = new(Config)

func main() {
	home := os.Getenv("HOME")
	config_path := flag.String("c", home+"/.godropbox.conf", "set configuration file `path`")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Gdbox is a command line tool for managing dropbox")
		fmt.Fprintln(os.Stderr, "Usage:\n\n\tgdbox [flags] command [arguments...]\n")
		fmt.Fprintln(os.Stderr, "The commands and arguments are:\n")
		fmt.Fprintln(os.Stderr, "\tdownload [src] [dst]\t\tdownload files/folders from dropbox")
		fmt.Fprintln(os.Stderr, "\tupload [src] [dst]\t\tupload files/folders to dropbox")
		fmt.Fprintln(os.Stderr, "\tfind [path] [expression]\tsearch for files in dropbox")
		fmt.Fprintln(os.Stderr, "\tmv [src] [dst]\t\t\tmove files")
		fmt.Fprintln(os.Stderr, "\tcp [src] [dst]\t\t\tcopy files")
		fmt.Fprintln(os.Stderr, "\tmkdir [path]\t\t\tmake a folder")
		fmt.Fprintln(os.Stderr, "\tls [file]\t\t\tlist files/folders in dropbox")
		fmt.Fprintln(os.Stderr, "\trm [file]\t\t\tdelete files")
		fmt.Fprintln(os.Stderr, "\nThe (optional) flags are:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if nil != flag.CommandLine.Parse(os.Args[1:]) {
		return
	}
	if flag.NArg() == 0 {
		flag.Usage()
		return
	} else {
		err := kConfig.LoadFile(*config_path)
		defer kConfig.SaveFile(*config_path)
		if err != nil {
			fmt.Println(err)
			return
		}
		if kConfig.AccessToken == "" {
			token := Setup()
			kConfig.AccessToken = token.AccessToken
		}
		handler(flag.CommandLine)
	}
}

type Config struct {
	AccessToken string `json:"access_token"`
}

func (c *Config) ToToken() *Token {
	token := Token{
		AccessToken: c.AccessToken,
	}
	return &token
}

// Change hanlder to handler -> handlerdownlaod, handler upload etc...
func handler(flag *flag.FlagSet) {
	command := flag.Arg(0)
	client := &http.Client{}
	switch command {
	case "download":
		if !(flag.NArg() == 2 || flag.NArg() == 3) {
			fmt.Println("Illegal number of arguments.\nTry " + os.Args[0] + " -h for more information")
			return
		}
		default_argument := flag.Arg(2)
		if flag.NArg() == 2 {
			default_argument = "."
		}
		metadata, err := GetMetadata(client, flag.Arg(1), kConfig.ToToken())
		if err != nil {
			fmt.Println(err)
			return
		}
		if metadata.IsDir {
			file_list := metadata.FileList(100, 0)
			for _, file := range file_list {
				fmt.Println("Downloading " + file + " to " + default_argument + file)
				err := Download(client, file, default_argument+file, kConfig.ToToken())
				if err != nil {
					fmt.Println(err)
				}
			}
		} else {
			fmt.Println("Downloading " + flag.Arg(1) + " to " + default_argument)
			err := Download(client, flag.Arg(1), default_argument, kConfig.ToToken())
			if err != nil {
				fmt.Println(err)
			}
		}
	case "upload":
		if flag.NArg() != 3 {
			fmt.Println("Illegal number of arguments.\nTry " + os.Args[0] + " -h for more information")
			return
		}
		err := Upload(client, flag.Arg(2), flag.Arg(1), kConfig.ToToken())
		if err != nil {
			fmt.Println(nil)
			return
		}
	case "find":
		if flag.NArg() != 3 {
			fmt.Println("Illegal number of arguments.\nTry " + os.Args[0] + " -h for more information")
			return
		}
		metadata, err := Search(client, flag.Arg(1), flag.Arg(2), kConfig.ToToken())
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, m := range metadata {
			fmt.Println(m.Path)
		}
	case "ls":
		if !(flag.NArg() == 2) {
			fmt.Println("Illegal number of arguments.\nTry " + os.Args[0] + " -h for more information")
			return
		}
		default_argument := flag.Arg(1)
		if flag.NArg() == 1 {
			default_argument = "."
		}
		metadata, err := GetMetadata(client, default_argument, kConfig.ToToken())
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Print(metadata.FormatFileNames())
	/*case "shell":
	fmt.Println("Starting Dropbox shell...")*/
	case "mv":
		if flag.NArg() != 2 {
			fmt.Println("Illegal number of arguments.\nTry " + os.Args[0] + " -h for more information")
			return
		}
		_, err := Move(client, flag.Arg(1), flag.Arg(2), kConfig.ToToken())
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Move operation successful")
	case "cp":
		if flag.NArg() != 2 {
			fmt.Println("Illegal number of arguments.\nTry " + os.Args[0] + " -h for more information")
			return
		}
		_, err := Copy(client, flag.Arg(1), flag.Arg(2), kConfig.ToToken())
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Copy operation successful")
	case "rm":
		if flag.NArg() != 2 {
			fmt.Println("Illegal number of arguments.\nTry " + os.Args[0] + " -h for more information")
			return
		}
		fmt.Print("Are you sure you wan't to delete " + flag.Arg(1) + "(y/n)")
		var answer string
		fmt.Scanf("%c", &answer)
		if strings.ToLower(answer) == "y" {
			_, err := Delete(client, flag.Arg(1), kConfig.ToToken())
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		fmt.Println("Delete operation successful")
	case "mkdir":
		if flag.NArg() != 2 {
			fmt.Println("Illegal number of arguments.\nTry " + os.Args[0] + " -h for more information")
			return
		}
		_, err := CreateFolder(client, flag.Arg(1), kConfig.ToToken())
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Mkdir operation successful")
	default:
		fmt.Println("Illegal command:" + command)
		fmt.Println("Try " + os.Args[0] + " -h for more information")
	}
}

func (c *Config) SaveFile(config_path string) error {
	output, _ := json.Marshal(c)
	err := ioutil.WriteFile(config_path, output, 600)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) LoadFile(config_path string) error {
	f, err := os.Open(config_path)
	if err != nil {
		if os.IsNotExist(err) {
			//fmt.Println("Configuration file does not exist, making one instead")
			return errors.New("Configuration file does not exist")
			//output, _ := json.Marshal(kConfig)
			//ioutil.WriteFile(config_path, output, 600)
		} else {
			return errors.New("Error opening configuration file:" + err.Error())
		}
	}
	scanner := bufio.NewScanner(f)
	var config_json []byte
	for scanner.Scan() {
		if strings.HasPrefix(strings.TrimSpace(scanner.Text()), "#") {
			continue
		} else {
			config_json = append(config_json, scanner.Bytes()...)
		}
	}
	err = json.Unmarshal(config_json, c)
	if err != nil {
		return errors.New("Config file has illegal syntax: " + err.Error())
	}
	return nil
}

func Setup() Token {
	var (
		app_key    string
		secret_key string
	)

	fmt.Println("You need to complete this setup inorder to use this program")
	fmt.Println("Open the following URL in your browser and login:", "https://www.dropbox.com/developers/apps/create")
	fmt.Println("Choose 'Dropbox Api App' and then choose the appropriate permissions\n and access restriction you want to apply.")
	fmt.Println("Enter any app name you want.")
	fmt.Println("After you sucessfully created your own app. Go to the app configuration page.")
	fmt.Println("Please type the app key and secret key below.")
	for {
		fmt.Print("App Key:")
		fmt.Scanln(&app_key)
		fmt.Print("App Secret:")
		fmt.Scanln(&secret_key)
		fmt.Printf("The App Key is %v and App Secret Key is %v. It is Ok?(y/n)\n", app_key, secret_key)
		var answer string
		fmt.Scanf("%c", &answer)
		if strings.ToLower(answer) == "y" {
			break
		}
	}
	authorize_url := "https://www.dropbox.com/1/oauth2/authorize"
	response_type := "code"
	authorize_url = authorize_url + "?" + "response_type=" + response_type + "&client_id=" + app_key
	fmt.Printf("Now open the following URL in your browser: %s\n", authorize_url)
	var auth_code string
	for {
		fmt.Println("Please type the given authorization code")
		fmt.Scanf("%s", &auth_code)
		fmt.Printf("The Authoirzation code is %v. Is it Ok?(y/n)", auth_code)
		var answer string
		fmt.Scanf("%c", &answer)
		if strings.ToLower(answer) == "y" {
			break
		}
	}
	token, _ := Oath2Athorize(app_key, secret_key, auth_code)
	return token
}

// Rename this function and change parameter
// name: such as subdirectory
//so we do not neet to input file_limit
func GetSubfileNames(path string, file_limit int) []string {
	var files []string
	parent, err := os.Open(path)
	defer parent.Close()
	if err != nil {
		return files
	}
	if stat, _ := parent.Stat(); !stat.IsDir() {
		return []string{path}
	}
	queue, _ := parent.Readdirnames(0)
	path = strings.TrimSuffix(path, "/")
	for idx, name := range queue {
		queue[idx] = path + "/" + name
	}
	//    queue = append(queue, path)
	for len(queue) > 0 {
		head := queue[0]
		if len(files) > file_limit {
			return files
		}
		child, err := os.Open(head)
		if err != nil {
			queue = queue[1:]
			continue
		}
		if child_info, _ := child.Stat(); child_info.IsDir() {
			child_files, _ := child.Readdirnames(0)
			for idx, name := range child_files {
				child_files[idx] = head + "/" + name
			}
			queue = append(queue, child_files...)
			queue = queue[1:]
		} else {
			files = append(files, head)
			queue = queue[1:]
		}
		child.Close()
	}
	return files
}
