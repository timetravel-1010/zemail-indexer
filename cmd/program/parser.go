package program

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	regexEmailAddress = `[\w._%+-]+@[\w.-]+\.[A-Za-z]{2,}`
	regexName         = `^[a-zA-ZÀ-ÿ0-9 ()-]*$`
)

type Email struct {
	MessageID               string   `json:"Message-ID"`
	Date                    string   `json:"Date"`
	From                    string   `json:"From"`
	To                      []string `json:"To"`
	CC                      []string `json:"CC"`
	BCC                     []string `json:"BCC"`
	Subject                 string   `json:"Subject"`
	MimeVersion             string   `json:"Mime-Version"`
	ContentType             string   `json:"Content-Type"`
	ContentTransferEncoding string   `json:"Content-Transfer-Encoding"`
	XFrom                   string   `json:"X-From"`
	XTo                     []string `json:"X-To"`
	Xcc                     []string `json:"X-cc"`
	Xbcc                    []string `json:"X-bcc"`
	XFolder                 string   `json:"X-Folder"`
	XOrigin                 string   `json:"X-Origin"`
	XFileName               string   `json:"X-FileName"`
	Body                    string   `json:"Body"`
}

type Document struct {
	Path  string `json:"path"`
	Email *Email `json:"email"`
}

// Parse
func Parse(fileName string) (*Email, error) {
	em := Email{}

	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentField := ""

	for scanner.Scan() {
		line := scanner.Text()
		subStrings := strings.SplitN(line, ":", 2)
		if len(subStrings) == 2 && currentField != "X-FileName" {
			currentField = subStrings[0]
			l := strings.TrimSpace(subStrings[1])
			switch currentField {
			case "Message-ID":
				//fmt.Println("Matched MessageID")
				em.MessageID = l
			case "Date":
				//fmt.Println("Matched Date")
				em.Date = l
			case "From":
				//fmt.Println("Matched From")
				em.From = l
			case "To":
				//fmt.Println("Matched To")
				em.To = parseAddresses(l)
				em.To = append(em.To, parseNames(l)...)
			case "Cc":
				//fmt.Println("Matched CC")
				em.CC = parseAddresses(l)
				em.CC = append(em.CC, parseNames(l)...)
				//fmt.Println(em.CC)
			case "Bcc":
				//fmt.Println("Matched BCC")
				em.BCC = parseAddresses(l)
				em.BCC = append(em.BCC, parseNames(l)...)
			case "Subject":
				//fmt.Println("Matched Subject")
				em.Subject = l
			case "Mime-Version":
				em.MimeVersion = l
				//fmt.Println("Matched Mime-Version")
			case "Content-Type":
				em.ContentType = l
				//fmt.Println("Matched Content-Type")
			case "Content-Transfer-Encoding":
				//fmt.Println("Matched Content-Transfer-Encoding")
				em.ContentTransferEncoding = l
			case "X-From":
				em.XFrom = l
				//fmt.Println("Matched X-From")
			case "X-To":
				//fmt.Println("Matched X-To")
				em.XTo = MapStrings(strings.Split(l, ","), strings.TrimSpace)
			case "X-cc":
				em.Xcc = parseAddresses(l)
				em.Xcc = append(em.Xcc, parseNames(l)...)
				//fmt.Println("Matched X-cc")
				//fmt.Println("Xcc len:", len(em.Xcc))
			case "X-bcc":
				em.Xbcc = parseAddresses(l)
				em.Xbcc = append(em.Xbcc, parseNames(l)...)
				//fmt.Println("Matched X-bcc")
			case "X-Folder":
				em.XFolder = l
				//fmt.Println("Matched X-Folder")
			case "X-Origin":
				em.XOrigin = l
				//fmt.Println("Matched X-Origin")
			case "X-FileName":
				em.XFileName = l
				//fmt.Println("Matched X-FileName")
			default:
				fmt.Println("No match found and currentLine=", currentField)
			}
		} else if currentField == "X-FileName" { // Body content
			em.Body += "\n"
			if subStrings != nil {
				em.Body += subStrings[0]
			}
		} else if currentField == "To" {
			em.To = append(em.To, parseAddresses(subStrings[0])...)
		}
	}
	//fmt.Printf("%+v\n", em)
	return &em, nil
}

func Index(dir string, re HttpRequest) {
	var counter int = 0
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	//encoder.Encode(IndexAction{Index: IndexDocument{Index: "enronmail"}})
	emails := []Document{}
	log.Println("Indexing documents...")
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			em, err := Parse(path)
			if err != nil {
				return err
			}
			emails = append(emails, Document{Path: path, Email: em})
			// encoder.Encode(em)
			counter++
			if counter == 100 {
				encoder.Encode(Payload{Index: re.Index, DocumentData: emails})
				Upload(re, buf)
				buf.Reset()
				counter = 0
				// encoder.Encode(IndexAction{Index: IndexDocument{Index: "enronmail"}})
			}
			//folders := strings.Split(path, "/")
			//fmt.Println("fileInfo:", folders) //folders[:len(folders)-1])
		}
		return nil
	})
	if err != nil {
		panic("Error while opening the files!")
	}
	if counter > 0 {
		encoder.Encode(Payload{Index: re.Index, DocumentData: emails})
		Upload(re, buf)
	}
	log.Println("Indexing completed successfully completed.")
}

func parseAddresses(s string) []string {
	return GetStringsByRegexp(s, regexEmailAddress)
}

func parseNames(s string) []string {
	return GetStringsByRegexp(s, regexName)
}

func GetStringsByRegexp(s string, regex string) []string {
	return regexp.MustCompile(regex).FindAllString(s, -1)
}

func print(s []string) {
	fmt.Print("[")
	for _, v := range s {
		if v == "\n" {
			fmt.Print("newline, ")
		} else if v == "" {
			fmt.Print("nil, ")
		} else if v == " " {
			fmt.Print("empty, ")
		} else {
			fmt.Print(v, ", ")
		}
	}
	fmt.Println("]")
}

func MapStrings(arr []string, f func(string) string) []string {
	newArr := make([]string, len(arr))
	for i, s := range arr {
		newArr[i] = f(s)
	}
	return newArr
}
