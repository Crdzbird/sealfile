package sealfile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// IsImageFile checks if a file is an image based on extension
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	pattern := `\.(jpg|jpeg|png|gif|bmp|webp|tiff|tif|svg|ico|heic|heif|jfif|pjpeg|pjp|raw|arw|cr2|nrw|k25|dng|nef|orf|raf|rw2|pef|sr2|srf|srw|3fr|mef|mos|bay|iiq|rwl|erf|mrw|x3f|svgz|indd|ai|eps|apng|avif|j2k|jp2|jpf|jpm|jpx|jxr|wdp|cur|emf|wmf|dds|icns|qoi|ras|sgi|sun|tga|xbm|xpm)$`
	matched, err := regexp.MatchString(pattern, ext)
	if err != nil {
		return false
	}
	return matched
}

// IsVideoFile checks if a file is a video based on extension
func IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	pattern := `\.(mp4|mov|avi|mkv|webm|flv|wmv|m4v|mpg|mpeg|3gp|3g2|mts|m2ts|ts|vob|ogv|rm|rmvb|asf|amv|divx|f4v|mxf|roq|svi|viv|yuv|mpe|mpv|qt|dat|drc|gifv|mng|nsv|nut|ogm|str|svi|vob|wtv|xesc)$`
	matched, err := regexp.MatchString(pattern, ext)
	if err != nil {
		return false
	}
	return matched
}

// IsAudioFile checks if a file is an audio file based on extension
func IsAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	pattern := `\.(mp3|wav|flac|aac|ogg|wma|m4a|aiff|alac|amr|ape|au|dss|dvf|gsm|iklax|ivs|m4b|m4p|mmf|mpc|msv|oga|opus|ra|rm|raw|sln|tta|voc|vox|wv|8svx|cda|mid|midi|mka|mpa|mp2|mp1|spx|ac3|adx|caf|mogg|mod|s3m|xm|it|stm|mtm|umx|kar|ram|shn|wv|xa|xhe|dsf|dff|rf64|sd2|ivs|ivs2|ivs3|ivs4|ivs5|ivs6|ivs7|ivs8|ivs9|ivs10|ivs11|ivs12|ivs13|ivs14|ivs15|ivs16|ivs17|ivs18|ivs19|ivs20|ivs21|ivs22|ivs23|ivs24|ivs25|ivs26|ivs27|ivs28|ivs29|ivs30|ivs31|ivs32|ivs33|ivs34|ivs35|ivs36|ivs37|ivs38|ivs39|ivs40|ivs41|ivs42|ivs43|ivs44|ivs45|ivs46|ivs47|ivs48|ivs49|ivs50)$`
	matched, err := regexp.MatchString(pattern, ext)
	if err != nil {
		return false
	}
	return matched
}

// IsDocumentFile checks if a file is a document based on extension
func IsDocumentFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	pattern := `\.(pdf|doc|docx|dot|dotx|dotm|docm|txt|rtf|odt|ott|fodt|pages|xls|xlsx|xlsm|xlsb|xlt|xltx|xltm|csv|tsv|ods|ots|fods|ppt|pptx|pptm|pps|ppsx|ppsm|pot|potx|potm|odp|otp|fodp|md|epub|mobi|azw|azw3|fb2|djvu|tex|log|wpd|wps|sxi|sti|sldx|sldm|key|numbers|one|xps|oxps|abw|sdw|stw|sxw|uot|uof|hwp|602|ps|psd|indd|pub|pmd|cwk|lwp|wp|wp5|wp6|wp7|wks|wdb|wri|prn|dvi|gdoc|gslides|gsheet|gdraw|gform|gtable|gslides|gdoc|gslides|gdraw|gform|gtable)$`
	matched, err := regexp.MatchString(pattern, ext)
	if err != nil {
		return false
	}
	return matched
}

// CreateTempFile creates a temporary file with the given data
func CreateTempFile(dir, filename string, data []byte) (*os.File, error) {
	tempName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename)
	tempPath := filepath.Join(dir, tempName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	file, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		err := file.Close()
		if err != nil {
			return nil, err
		}
		err = os.Remove(tempPath)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to write data to temp file: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		err := file.Close()
		if err != nil {
			return nil, err
		}
		err = os.Remove(tempPath)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	return file, nil
}

// GetFileNameWithoutExtension returns filename without extension
func GetFileNameWithoutExtension(filename string) string {
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}

// GetFileExtension returns the file extension (including the dot)
func GetFileExtension(filename string) string {
	return filepath.Ext(filename)
}

// SanitizeFilename removes or replaces invalid characters in filename
func SanitizeFilename(filename string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename
	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}
	sanitized = strings.Trim(sanitized, " .")
	if sanitized == "" {
		return "untitled"
	}
	return sanitized
}

// EnsureDirectory creates directory if it doesn't exist
func EnsureDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filepath string) (int64, error) {
	info, err := os.Stat(filepath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}
	return info.Size(), nil
}
