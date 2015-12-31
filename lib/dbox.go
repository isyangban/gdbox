package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

//TODO: Set Some basic consttants
type dboxConst struct {
	MaxFileLimit          int
	DirectUploadSizeLimit int
}

var kDboxConst = dboxConst{
	MaxFileLimit:          10000,
	DirectUploadSizeLimit: 15 * 1000 * 1000,
}

type Dropbox struct {
	Account  Account
	Token    Token
	Metadata map[string]Metadata
	Client   *http.Client
}

func NewDropbox(token Token) *Dropbox {
	dbox := &Dropbox{}
	dbox.Token = token
	dbox.Client = &http.Client{}
	dbox.Metadata = make(map[string]Metadata)
	return dbox
}
func (dbox *Dropbox) AddAuthHeader(r *http.Request) {
	//var auth_header = "Bearer <YOUR_ACCESS_TOKEN_HERE>"
	r.Header.Add("Authorization", "Bearer "+dbox.Token.AccessToken)
}

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Uid         string `json:"uid"`
}

func NewToken(token_json []byte) *Token {
	var token Token
	json.Unmarshal(token_json, &token)
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

func (dbox *Dropbox) GetAccount() (Account, error) {
	req, _ := http.NewRequest("GET", "https://api.dropbox.com/1/account/info", nil)
	dbox.AddAuthHeader(req)
	resp, _ := dbox.Client.Do(req)
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		body, _ := ioutil.ReadAll(resp.Body)
		dbox.Account = *NewAccount(body)
		return dbox.Account, nil
	default:
		return Account{}, errors.New(resp.Status)
	}
}

func (dbox *Dropbox) Oath2Athorize(client_id, client_secret, auth_code string) error {
	parm := url.Values{"code": {auth_code}, "grant_type": {"authorization_code"}, "client_id": {client_id}, "client_secret": {client_secret}}
	resp, err := http.PostForm("https://api.dropbox.com/1/oauth2/token", parm)
	if err != nil {
		return errors.New("Error in recieving token\n")
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		body, _ := ioutil.ReadAll(resp.Body)
		dbox.Token = *NewToken(body)
		return nil
	default:
		return errors.New(resp.Status)
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

func NewMetadata(metadata_json []byte) *Metadata {
	var metadata Metadata
	json.Unmarshal(metadata_json, &metadata)
	return &metadata
}

//Change name to File name list
//Used for getting all the file names in a folder
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
		result = m.Path[strings.LastIndex(m.Path, "/"):]
	} else {
		for _, c := range m.Contents {
			file_names = append(file_names, c.Path[strings.LastIndex(c.Path, "/")+1:])
		}
		result = Format(file_names)
	}
	return result
}

//TODO: Needs some exception handling when metadata is an empty struct
func (dbox *Dropbox) GetMetaData(filepath string) (Metadata, error) {
	parm := url.Values{"file_limit": {"100"}, "list": {"true"}, "include_deleted": {"false"}, "include_media_info": {"false"}, "include_membership": {"false"}}
	if dbox.Metadata[filepath].Hash != "" {
		parm.Add("hash", dbox.Metadata[filepath].Hash)
	}
	url_path := strings.Replace(url.QueryEscape(filepath), "+", "%20", -1)
	req, _ := http.NewRequest("GET", "https://api.dropbox.com/1/metadata/auto/"+url_path+"?"+parm.Encode(), nil)
	dbox.AddAuthHeader(req)
	resp, _ := dbox.Client.Do(req)
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		body, _ := ioutil.ReadAll(resp.Body)
		metadata := *NewMetadata(body)
		if metadata.IsDir {
			dbox.Metadata[filepath] = metadata
		}
		return metadata, nil
	case 304:
		return dbox.Metadata[filepath], nil
	case 404:
		return Metadata{}, errors.New("File " + filepath + " not found")
	case 406:
		return Metadata{}, errors.New("Too many files to return")
	default:
		return Metadata{}, errors.New(resp.Status)
	}
}

func (dbox *Dropbox) Copy(from_path string, to_path string) (Metadata, error) {
	parm := url.Values{"root": {"auto"}, "from_path": {from_path}, "to_path": {to_path}}
	req, _ := http.NewRequest("POST", "https://api.dropbox.com/1/fileops/copy", strings.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dbox.AddAuthHeader(req)
	resp, _ := dbox.Client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	switch resp.StatusCode {
	case 200:
		return *NewMetadata(body), nil
	default:
		return Metadata{}, errors.New(string(body))
	}
}

func (dbox *Dropbox) Move(from_path string, to_path string) (Metadata, error) {
	parm := url.Values{"root": {"auto"}, "from_path": {from_path}, "to_path": {to_path}}
	req, _ := http.NewRequest("POST", "https://api.dropbox.com/1/fileops/move", strings.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dbox.AddAuthHeader(req)
	resp, _ := dbox.Client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	switch resp.StatusCode {
	case 200:
		return *NewMetadata(body), nil
	default:
		return Metadata{}, errors.New(string(body))
	}
}

func (dbox *Dropbox) CreateFolder(dir_path string) (Metadata, error) {
	parm := url.Values{"root": {"auto"}, "path": {dir_path}}
	req, _ := http.NewRequest("POST", "https://api.dropbox.com/1/fileops/create_folder", strings.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dbox.AddAuthHeader(req)
	resp, _ := dbox.Client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	switch resp.StatusCode {
	case 200:
		return *NewMetadata(body), nil
	case 403:
		return Metadata{}, errors.New("There is already a folder at the given destination")
	default:
		return Metadata{}, errors.New(string(body))
	}
}

func (dbox *Dropbox) Search(folder_path string, query string) ([]Metadata, error) {
	parm := url.Values{"file_limit": {"1000"}, "include_deleted": {"false"}}
	for _, word := range strings.Split(query, " ") {
		parm.Add("query", word)
	}
	url_path := strings.Replace(url.QueryEscape(folder_path), "+", "%20", -1)
	req, _ := http.NewRequest("GET", "https://api.dropboxapi.com/1/search/auto/"+url_path+"?"+parm.Encode(), nil)
	dbox.Client.Do(req)
	metadata_list := make([]Metadata, 100) //What if unmarshal requires more size?
	body, _ := ioutil.ReadAll(resp.Body)
	err := json.Unmarshal(body, &metadata_list)
	if err != nil {
		return make([]Metadata, 0), err
	}
	return metadata_list, nil
}

func (dbox *Dropbox) Delete(path string) (Metadata, error) {
	parm := url.Values{"root": {"auto"}, "path": {path}}
	req, _ := http.NewRequest("POST", "https://api.dropbox.com/1/fileops/delete", strings.NewReader(parm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//req.ParseForm()
	dbox.AddAuthHeader(req)
	resp, _ := dbox.Client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	switch resp.StatusCode {
	case 200:
		return *NewMetadata(body), nil
	default:
		return Metadata{}, errors.New(resp.Status)
	}
}

//TODO: If there is no folders then make folders first
func (dbox *Dropbox) Download(remote_path string, local_path string) error {
	//parm := url.Values{"rev"}
	url_path := strings.Replace(url.QueryEscape(remote_path), "+", "%20", -1)
	req, _ := http.NewRequest("GET", "https://api-content.dropbox.com/1/files/auto/"+url_path, nil)
	dbox.AddAuthHeader(req)
	resp, _ := dbox.Client.Do(req)
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 404:
		return errors.New("File " + remote_path + " is not found on dropbox")
	case 200:
		metadata_json := resp.Header.Get("x-dropbox-metadata")
		metadata := NewMetadata([]byte(metadata_json)) // Another way arround?
		body, _ := ioutil.ReadAll(resp.Body)
		if len(body) != metadata.Bytes {
			fmt.Println("Download size does not match, download: ", len(body), " expected: ", metadata.Bytes)
		}
		// Need complete rewrite using path.filepath
		// user os.MkdriAll and ioutil.WriteFile(x, x, 0644)
		os.MkdirAll(filepath.Dir(local_path), 0755)
		stat, err := os.Stat(local_path)
		if err != nil {
			if os.IsNotExist(err) {
				err := ioutil.WriteFile(local_path, body, 0644)
				if err != nil {
					return err
				}
				return err
			}
		}
		if stat.IsDir() {
			err := ioutil.WriteFile(local_path+filepath.Base(remote_path), body, 0644)
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
	default:
		return errors.New(resp.Status)
	}
}
