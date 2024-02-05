package program

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/timetravel-1010/indexer/cmd/util"
)

var (
	fields = []string{
		"Message-ID",
		"Date",
		"From",
		"To",
		"Cc",
		"Bcc",
		"Subject",
		"Mime-Version",
		"Content-Type",
		"Content-Transfer-Encoding",
		"X-From",
		"X-To",
		"X-cc",
		"X-bcc",
		"X-Folder",
		"X-Origin",
		"X-FileName",
		"Body",
	}
)

const (
	regexEmailAddress = `[\w._%+-]+@[\w.-]+\.[A-Za-z]{2,}`
	regexName         = `^[a-zA-ZÀ-ÿ0-9 ()-]*$`
)

// An Email contains all the information of an e-mail.
type Email struct {
	MessageID               string   `json:"messageId"`
	Date                    string   `json:"date"`
	From                    string   `json:"from"`
	To                      []string `json:"to"`
	CC                      []string `json:"cc"`
	BCC                     []string `json:"bcc"`
	Subject                 string   `json:"subject"`
	MimeVersion             string   `json:"mimeVersion"`
	ContentType             string   `json:"contentType"`
	ContentTransferEncoding string   `json:"contentTransferEncoding"`
	XFrom                   string   `json:"xFrom"`
	XTo                     []string `json:"xTo"`
	Xcc                     []string `json:"xcc"`
	Xbcc                    []string `json:"xbcc"`
	XFolder                 string   `json:"xFolder"`
	XOrigin                 string   `json:"xOrigin"`
	XFileName               string   `json:"xFileName"`
	Body                    string   `json:"body"`
}

// A Document contains the path of the email and the email itself.
type Document struct {
	Path  string `json:"path"` // path to the email.
	Email *Email `json:"email"`
}

// A Parser
type Parser struct {
}

// Parse parses the txt email file into the Email structure.
// If there is an error, it will be of type *PathError.
func (p *Parser) Parse(filePath string) (*Email, error) {
	em := Email{}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentField := ""
	inBody := false

	for scanner.Scan() {
		line := scanner.Text()
		if isValid := saveLine(&em, line, &currentField, filePath, &inBody); !isValid {
			break
		}
	}
	return &em, nil
}

func CheckEmpty(filePath string) (bool, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return true, err
	}

	return (fi.Size() == 0), nil
}

// parseAddresses
func parseAddresses(s string) []string {
	return GetStringsByRegexp(s, regexEmailAddress)
}

// parseNames
func parseNames(s string) []string {
	return GetStringsByRegexp(s, regexName)
}

// GetStringsByRegexp
func GetStringsByRegexp(s string, regex string) []string {
	return regexp.MustCompile(regex).FindAllString(s, -1)
}

// MapStrings
func MapStrings(arr []string, f func(string) string) []string {
	newArr := make([]string, len(arr))
	for i, s := range arr {
		newArr[i] = f(s)
	}
	return newArr
}

func saveLine(em *Email, line string, currentField *string, filePath string, inBody *bool) bool {
	if *inBody {
		em.Body += fmt.Sprintf("\n%s", line)
		return true
	}
	subStrings := strings.SplitN(line, ":", 2)
	first := subStrings[0]

	if idx := util.IndexOf(first, fields); idx == -1 { // Continues in a section.
		addLine(line, *currentField, em, filePath)
	} else if len(subStrings) == 2 {
		*currentField = subStrings[0]
		line := strings.TrimSpace(subStrings[1])
		setValue(*currentField, line, em, filePath)
	}
	// TODO: Move this validation to do it just once.
	if em.MessageID == "" {
		log.Printf("The file %s is not an email, skipped.\n", filePath)
		return false
	}

	*inBody = *currentField == "X-FileName"
	return true
}

func setValue(currentField, l string, em *Email, filePath string) {
	switch currentField {
	case "Message-ID":
		em.MessageID = l
	case "Date":
		em.Date = l
	case "From":
		em.From = l
	case "To":
		em.To = parseAddresses(l)
		em.To = append(em.To, parseNames(l)...)
	case "Cc":
		em.CC = parseAddresses(l)
		em.CC = append(em.CC, parseNames(l)...)
	case "Bcc":
		em.BCC = parseAddresses(l)
		em.BCC = append(em.BCC, parseNames(l)...)
	case "Subject":
		em.Subject = l
	case "Mime-Version":
		em.MimeVersion = l
	case "Content-Type":
		em.ContentType = l
	case "Content-Transfer-Encoding":
		em.ContentTransferEncoding = l
	case "X-From":
		em.XFrom = l
	case "X-To":
		em.XTo = MapStrings(strings.Split(l, ","), strings.TrimSpace)
	case "X-cc":
		em.Xcc = parseAddresses(l)
		em.Xcc = append(em.Xcc, parseNames(l)...)
	case "X-bcc":
		em.Xbcc = parseAddresses(l)
		em.Xbcc = append(em.Xbcc, parseNames(l)...)
	case "X-Folder":
		em.XFolder = l
	case "X-Origin":
		em.XOrigin = l
	case "X-FileName":
		em.XFileName = l
	default:
		fmt.Println(fmt.Sprintf(`
        ===================ERROR NO MATCH FOUND
        function: setValue
        l: %s
        currentLine: %s
        file: %s
        ===================END ERROR`, l, currentField, filePath))
	}
}

func addLine(l, currentField string, em *Email, filePath string) {
	switch currentField {
	case "Message-ID":
		em.MessageID += l
	case "Date":
		em.Date += l
	case "From":
		em.From += l
	case "To":
		em.To = append(em.To, parseAddresses(l)...)
		em.To = append(em.To, parseNames(l)...)
	case "Cc":
		em.CC = parseAddresses(l)
		em.CC = append(em.CC, parseNames(l)...)
	case "Bcc":
		em.BCC = parseAddresses(l)
		em.BCC = append(em.BCC, parseNames(l)...)
	case "Subject":
		em.Subject += "\n" + l
	case "Mime-Version":
		em.MimeVersion += l
	case "Content-Type":
		em.ContentType += l
	case "Content-Transfer-Encoding":
		em.ContentTransferEncoding += l
	case "X-From":
		em.XFrom += l
	case "X-To":
		em.XTo = append(em.XTo, MapStrings(strings.Split(l, ","), strings.TrimSpace)...)
	case "X-cc":
		em.Xcc = parseAddresses(l)
		em.Xcc = append(em.Xcc, parseNames(l)...)
	case "X-bcc":
		em.Xbcc = parseAddresses(l)
		em.Xbcc = append(em.Xbcc, parseNames(l)...)
	case "X-Folder":
		em.XFolder += l
	case "X-Origin":
		em.XOrigin += l
	case "X-FileName":
		em.XFileName += l
	default:
		fmt.Println("addLine: No match found and currentLine =", currentField, "file:", filePath)
	}
}
