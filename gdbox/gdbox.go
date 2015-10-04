package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	s "strings"
	"syscall"
	"unicode"
	"unsafe"
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
		if s.ToLower(answer) == "y" {
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
		if s.HasPrefix(s.TrimSpace(scanner.Text()), "#") {
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

func Oath2Athorize(client_id, client_secret, auth_code string) (Token, error) {
	parm := url.Values{"code": {auth_code}, "grant_type": {"authorization_code"}, "client_id": {client_id}, "client_secret": {client_secret}}
	resp, err := http.PostForm("https://api.dropbox.com/1/oauth2/token", parm)
	if err != nil {
		fmt.Println("Error in recieving token:", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	//	println(string(body[:]))
	token := NewToken(body)
	return *token, nil
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
		if s.ToLower(answer) == "y" {
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
		if s.ToLower(answer) == "y" {
			break
		}
	}
	token, _ := Oath2Athorize(app_key, secret_key, auth_code)
	return token
}

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Uid         string `json:"uid"`
}

func NewToken(token_json []byte) *Token {
	var token Token
	err := json.Unmarshal(token_json, &token)
	_ = err
	return &token
}

func (c *Config) ToToken() *Token {
	token := Token{
		AccessToken: c.AccessToken,
	}
	return &token
}

type Account struct {
	Display_name string `json:"display_name"`
	Uid          int    `json:"uid"`
	Locale       string `json:"locale"`
}

func NewAccount(account_json []byte) *Account {
	var account Account
	err := json.Unmarshal(account_json, &account)
	if err != nil {
		fmt.Println("Account Json Parse Error:", err)
	}
	return &account
}

func AddAuthHeader(r *http.Request, token *Token) {
	//var auth_header = "Bearer <YOUR_ACCESS_TOKEN_HERE>"
	r.Header.Add("Authorization", "Bearer "+token.AccessToken)
}

func GetAccount(c *http.Client, token *Token) (Account, error) {
	req, err := http.NewRequest("GET", "https://api.dropbox.com/1/account/info", nil)
	_ = err
	AddAuthHeader(req, token)
	resp, err := c.Do(req)
	_ = err
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return *NewAccount(body), nil
}

//TODO: Needs some exception handling when metadata is an empty struct
func GetMetadata(c *http.Client, filepath string, token *Token) (Metadata, error) {
	parm := url.Values{"file_limit": {"100"}, "list": {"true"}, "include_deleted": {"false"}, "include_media_info": {"false"}, "include_membership": {"false"}}
	if kMetadata[filepath].Hash != "" {
		parm.Add("hash", kMetadata[filepath].Hash)
	}
	url_path := s.Replace(url.QueryEscape(filepath), "+", "%20", -1)
	req, err := http.NewRequest("GET", "https://api.dropbox.com/1/metadata/auto/"+url_path+"?"+parm.Encode(), nil)
	AddAuthHeader(req, token)
	resp, err := c.Do(req)
	if err != nil {
		return Metadata{}, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 304:
		return kMetadata[filepath], nil
	case 406:
		return Metadata{}, errors.New("Too many files to return")
	case 404:
		return Metadata{}, errors.New("File " + filepath + " not found")
	case 200:
		body, _ := ioutil.ReadAll(resp.Body)
		metadata := *NewMetadata(body)
		if metadata.IsDir {
			kMetadata[filepath] = metadata
		}
		return metadata, nil
	default:
		return Metadata{}, errors.New(resp.Status)
	}
}

type Metadata struct {
	Size     string     `json:"size"`
	Rev      string     `json:"rev"`
	Bytes    int        `json:"bytes"`
	Modified string     `json:"modified"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"is_dir"`
	Root     string     `json:"root"`
	Revision int        `json:"revision"`
	Hash     string     `json:"hash"`
	Contents []Metadata `json:"contents"`
}

// Needs a more prettier format
func (m *Metadata) String() string {
	return "Path: " + m.Path
}

func NewMetadata(metadata_json []byte) *Metadata {
	var metadata Metadata
	json.Unmarshal(metadata_json, &metadata)
	return &metadata
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getWidth() uint {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		panic(errno)
	}
	return uint(ws.Col)
}

/*
func (m Metadata) Traverse(fn func(Metadata), depth int) {
	if depth == 1 {
		fn(m)
	} else if m.IsDir == false {
		fn(m)
	} else {
		for _, c := range m.Contents {
			c.Traverse(fn, depth+1)
		}
	}
}
*/

func (m *Metadata) FileList(file_limit int, acc int) []string {
	var files []string
	if !m.IsDir {
		acc += 1
		if acc > file_limit {
			return files
		} else {
			files = append(files, m.Path)
			return files
		}
	} else {
		for _, c := range m.Contents {
			acc += 1
			list := c.FileList(file_limit, acc)
			files = append(files, list...)
		}
		return files
	}
}

func (m *Metadata) FormatFileNames() string {
	var (
		result     string
		file_names []string
	)
	if m.IsDir == false {
		result = m.Path[s.LastIndex(m.Path, "/"):]
	} else {
		for _, c := range m.Contents {
			file_names = append(file_names, c.Path[s.LastIndex(c.Path, "/")+1:])
		}
		result = format(file_names)
	}
	return result
}
func format(file_names []string) string {
	term_width := int(getWidth())
	if term_width <= 0 {
		term_width = 100
	}
	total_width := 0
	tmp_col_length := []int{term_width}
	var col_length []int
	//fmt.Println("file_names", file_names)
	if length := MaxStringLength(file_names); length > term_width {
		col_length = append(col_length, length)
	} else {
		for col := 2; total_width <= term_width; col++ {
			row := rows(len(file_names), col)
			if row*(col-1) > len(file_names) {
				continue
			}
			col_length = make([]int, len(tmp_col_length))
			copy(col_length, tmp_col_length)
			tmp_col_length = make([]int, col)
			total_width = 0
			if row == 0 {
				break
			}
			for i := 0; i < col-1; i++ {
				tmp_col_length[i] = MaxStringLength(file_names[i*row:(i+1)*row]) + 2
				total_width += tmp_col_length[i] + 2
			}
			//fmt.Println(row, col, len(file_names))
			tmp_col_length[len(tmp_col_length)-1] = MaxStringLength(file_names[(col-1)*row:len(file_names)]) + 2
			total_width += tmp_col_length[len(tmp_col_length)-1] + 2
			//fmt.Println(total_width)
			//fmt.Println(tmp_col_length)
		}
	}
	col := len(col_length) //Adjust column
	var column string
	var columns []string
	length := col_length[0]
	row := rows(len(file_names), col)
	col_idx := 0
	for idx, file_name := range file_names {
		if idx%row == 0 && idx != 0 {
			columns = append(columns, column)
			column = ""
			col_idx++
			length = col_length[col_idx]
		}
		space_length := length - calcStringWidth(file_name)
		if space_length < 0 {
			fmt.Println(file_name)
			fmt.Println(length)
			fmt.Println(len(file_name))
		}
		column = column + file_name + s.Repeat(" ", space_length) + "\n"
	}
	columns = append(columns, column)
	left := columns[0]
	for _, c := range columns[1:] {
		left = CombineStr(left, "\n", c)
	}
	return left
}
func rows(items, col int) int {
	if items/col == 0 {
		return 0
	} else if items%col == 0 {
		return items / col
	} else {
		return items/col + 1
	}
}
func MaxStringLength(strlist []string) int {
	max := 0
	for _, str := range strlist {
		if calcStringWidth(str) > max {
			max = len(str)
		}
	}
	return max
}

func calcStringWidth(s string) int {
	width := 0
	for _, c := range s {
		if unicode.In(c, unicode.Hangul, unicode.Katakana, unicode.Hiragana, unicode.Han) {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

func CombineStr(left string, sep string, right string) string {
	left_lines := s.Split(left, sep)
	right_lines := s.Split(right, sep)
	var min int
	if len(left_lines) < len(right_lines) {
		min = len(left_lines)
	} else {
		min = len(right_lines)
	}
	for i := 0; i < min; i++ {
		left_lines[i] = left_lines[i] + right_lines[i]
	}
	return s.Join(left_lines, sep)
}

func Copy(c *http.Client, from_path string, to_path string, token *Token) (Metadata, error) {
	//strings.NewReader(form.Encode()))
	parm := url.Values{"root": {"auto"}, "from_path": {from_path}, "to_path": {to_path}}
	//fmt.Println(parm.Encode())
	req, err := http.NewRequest("POST", "https://api.dropbox.com/1/fileops/copy", s.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//req.ParseForm()
	AddAuthHeader(req, token)
	resp, err := c.Do(req)
	_ = err
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	return *NewMetadata(body), nil
}

func Move(c *http.Client, from_path string, to_path string, token *Token) (Metadata, error) {
	parm := url.Values{"root": {"auto"}, "from_path": {from_path}, "to_path": {to_path}}
	req, err := http.NewRequest("POST", "https://api.dropbox.com/1/fileops/move", s.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//req.ParseForm()
	AddAuthHeader(req, token)
	resp, err := c.Do(req)
	_ = err
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return *NewMetadata(body), nil
}

func CreateFolder(c *http.Client, dir_path string, token *Token) (Metadata, error) {
	parm := url.Values{"root": {"auto"}, "path": {dir_path}}
	req, err := http.NewRequest("POST", "https://api.dropbox.com/1/fileops/create_folder", s.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//req.ParseForm()
	AddAuthHeader(req, token)
	resp, err := c.Do(req)
	_ = err
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return *NewMetadata(body), nil
}

func Delete(c *http.Client, path string, token *Token) (Metadata, error) {
	parm := url.Values{"root": {"auto"}, "path": {path}}
	req, err := http.NewRequest("POST", "https://api.dropbox.com/1/fileops/delete", s.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//req.ParseForm()
	AddAuthHeader(req, token)
	resp, err := c.Do(req)
	_ = err
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return *NewMetadata(body), nil
}

//TODO: If there is no folders then make folders first
func Download(c *http.Client, remote_path string, local_path string, token *Token) error {
	//parm := url.Values{"rev"}
	url_path := s.Replace(url.QueryEscape(remote_path), "+", "%20", -1)
	req, _ := http.NewRequest("GET", "https://api-content.dropbox.com/1/files/auto/"+url_path, nil)
	AddAuthHeader(req, token)
	resp, _ := c.Do(req)
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return errors.New("File " + remote_path + " is not found on dropbox")
	}
	metadata_json := resp.Header.Get("x-dropbox-metadata")
	metadata := NewMetadata([]byte(metadata_json)) // Another way arround?
	body, _ := ioutil.ReadAll(resp.Body)
	if len(body) != metadata.Bytes {
		fmt.Println("Download size does not match, download: ", len(body), " expected: ", metadata.Bytes)
	}
	var folder_path string
	if idx := s.LastIndex(local_path, "/"); idx < 0 {
		folder_path = local_path
	} else {
		folder_path = local_path[:s.LastIndex(local_path, "/")]
	}
	os.MkdirAll(folder_path, 0755)
	info, err := os.Stat(local_path)
	if err != nil {
		if os.IsNotExist(err) {
			err := ioutil.WriteFile(local_path, body, 0644)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if info.IsDir() {
		fmt.Println(local_path, remote_path)
		err := ioutil.WriteFile(local_path+"/"+remote_path, body, 0644)
		if err != nil {
			return err
		}
	} else {
		err := ioutil.WriteFile(local_path, body, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func Upload(c *http.Client, remote_path string, local_path string, token *Token) error {
	//"Content-Type: application/octet-stream"?
	parm := url.Values{"overwrite": {"true"}, "autorename": {"true"}}
	files := bfsFolders(local_path, 100)
	for _, file := range files {
		f, err := os.Open(file)
		defer f.Close()
		if err != nil {
			fmt.Println("Error opening file: ", err)
			return err
		}
		file_stats, err := f.Stat()
		if file_stats.Size() > 20*1024*1024 {
			sf := io.NewSectionReader(f, 0, 20*1024*1024)
			upload_id, offset, _ := chunkedUpload(c, "", sf, 0, token)
			//Use Chunck Upload
			for sf.Size() > 0 {
				upload_id, offset, _ = chunkedUpload(c, upload_id, sf, offset, token)
				sf = io.NewSectionReader(f, 0, 20*1024*1024)
			}
			_, err := commitChunkedUpload(c, remote_path, upload_id, kConfig.ToToken())
			if err != nil {
				return err
			}
		} else {
			//Use Direct Upload
			req, _ := http.NewRequest("PUT", "https://api-content.dropbox.com/1/files_put/auto/"+url.QueryEscape(remote_path)+"?"+parm.Encode(), f)
			req.Header.Set("Content-Length", strconv.FormatInt(file_stats.Size(), 10))
			AddAuthHeader(req, token)
			//Need to set content-type?
			resp, _ := c.Do(req)
			//		body, _ := ioutil.ReadAll(resp.Body)
			//		println(body)if err != nil {
			defer resp.Body.Close()
			switch resp.StatusCode {
			case 200:
				return nil
			default:
				return errors.New(resp.Status)
			}
		}

	}
	return nil
}

type ChunkedFile struct {
	UploadId string `json:"upload_id"`
	Offset   int64  `json:"offset"`
	Expires  string `json:"expires"`
}

func chunkedUpload(c *http.Client, upload_id string, sf *io.SectionReader, offset int64, token *Token) (string, int64, error) {
	parm := url.Values{}
	if upload_id == "" {
		parm.Add("offset", "0")
	} else {
		parm.Add("offset", strconv.FormatInt(offset, 10))
		parm.Add("upload_id", upload_id)
	}
	req, _ := http.NewRequest("PUT", "https://api-content.dropbox.com/1/chunked_upload"+"?"+parm.Encode(), sf)
	AddAuthHeader(req, token)
	resp, _ := c.Do(req)
	chunked_file := new(ChunkedFile)
	body, _ := ioutil.ReadAll(resp.Body)
	err := json.Unmarshal(body, chunked_file)
	if err != nil {
		fmt.Println(err)
		return "", offset, err
	}
	return chunked_file.UploadId, chunked_file.Offset, nil
}

func commitChunkedUpload(c *http.Client, remote_path string, upload_id string, token *Token) (Metadata, error) {
	parm := url.Values{"upload_id": {upload_id}, "overwrite": {"true"}, "autorename": {"true"}}
	req, _ := http.NewRequest("POST", "https://content.dropboxapi.com/1/commit_chunked_upload/auto/"+url.QueryEscape(remote_path), s.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	AddAuthHeader(req, token)
	resp, _ := c.Do(req)
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 400:
		return Metadata{}, errors.New("Invalid upload id: " + upload_id + "or chunked file does not exist")
	case 409:
		return Metadata{}, errors.New("Conflict with existing file")
	case 200:
		body, _ := ioutil.ReadAll(resp.Body)
		return *NewMetadata(body), nil
	default:
		return Metadata{}, errors.New(resp.Status)
	}
}

func Search(c *http.Client, folder_path string, query string, token *Token) ([]Metadata, error) {
	parm := url.Values{"file_limit": {"1000"}, "include_deleted": {"false"}}
	for _, word := range s.Split(query, " ") {
		parm.Add("query", word)
	}
	url_path := s.Replace(url.QueryEscape(folder_path), "+", "%20", -1)
	req, _ := http.NewRequest("GET", "https://api.dropboxapi.com/1/search/auto/"+url_path+"?"+parm.Encode(), nil)
	AddAuthHeader(req, token)
	resp, _ := c.Do(req)
	metadata_list := make([]Metadata, 100) //What if unmarshal requires more size?
	body, _ := ioutil.ReadAll(resp.Body)
	err := json.Unmarshal(body, &metadata_list)
	if err != nil {
		return make([]Metadata, 0), err
	}
	return metadata_list, nil
}

//TODO: rename to SubfileNames
func bfsFolders(path string, file_limit int) []string {
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
	path = s.TrimSuffix(path, "/")
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
