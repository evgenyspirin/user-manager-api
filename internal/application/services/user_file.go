package services

import (
	"context"
	"fmt"
	"mime"
	"mime/multipart"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"user-manager-api/internal/application/ports"
	"user-manager-api/internal/domain/user"
	domain "user-manager-api/internal/domain/user_file"
)

const maxBaseNameLen = 100

var (
	windowsReserved = map[string]struct{}{
		"con": {}, "prn": {}, "aux": {}, "nul": {},
		"com1": {}, "com2": {}, "com3": {}, "com4": {}, "com5": {}, "com6": {}, "com7": {}, "com8": {}, "com9": {},
		"lpt1": {}, "lpt2": {}, "lpt3": {}, "lpt4": {}, "lpt5": {}, "lpt6": {}, "lpt7": {}, "lpt8": {}, "lpt9": {},
	}
	fileSafeRe    = regexp.MustCompile(`[^A-Za-z0-9\.\_\- ]+`)
	leadingDotsRe = regexp.MustCompile(`^\.+`)
)

type UserFileService struct {
	s3                 ports.S3Client
	userFileRepository domain.Repository
	userRepository     user.Repository
	mCounter           *prometheus.CounterVec
}

func NewUserFileService(
	s3 ports.S3Client,
	userFileRepository domain.Repository,
	userRepository user.Repository,
	mCounter *prometheus.CounterVec,
) ports.UserFileService {
	return &UserFileService{
		s3:                 s3,
		userFileRepository: userFileRepository,
		userRepository:     userRepository,
		mCounter:           mCounter,
	}
}

func (ufs *UserFileService) FindUserFiles(ctx context.Context, userUUID user.UUID, page int) (domain.UserFiles, error) {
	id, err := ufs.userRepository.FetchInternalID(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	fls, err := ufs.userFileRepository.FetchUserFiles(ctx, id, page)
	if err != nil {
		return nil, err
	}

	return fls, nil
}

func (ufs *UserFileService) CreateUserFile(
	ctx context.Context,
	userUUID user.UUID,
	in *multipart.FileHeader,
) (*domain.UserFile, error) {
	uf := new(domain.UserFile)

	id, err := ufs.userRepository.FetchInternalID(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	uf = ufs.fillMetaData(in, uf, userUUID)
	f, err := in.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// release memory
	f = nil

	// example: save obj to s3
	// ufs.s3.PutObject(...)

	out, err := ufs.userFileRepository.CreateUserFile(ctx, id, uf)
	if err != nil {
		return nil, err
	}

	ufs.mCounter.WithLabelValues("user_files_created_total").Inc()

	return out, nil
}

func (ufs *UserFileService) fillMetaData(
	in *multipart.FileHeader,
	uf *domain.UserFile,
	userUUID user.UUID,
) *domain.UserFile {
	uf.FileName = filepath.Base(sanitizeFileName(in.Filename))
	uf.MimeType = in.Header.Get("Content-Type")
	uf.SizeBytes = uint64(in.Size)
	uf.Bucket = ufs.s3.GetBucket()
	uf.StorageKey = ufs.genSafeStorageKey(uf, userUUID)
	uf.DownloadURL = ufs.s3.GetPublicURL(uf.StorageKey)

	return uf
}

// genSafeStorageKey: "documents/YYYY/MM/DD/<ts-nanosec>/<useruuid>/<filename>.ext"
func (ufs *UserFileService) genSafeStorageKey(
	uf *domain.UserFile,
	userUUID user.UUID,
) string {
	clean := strings.TrimSpace(uf.FileName)
	clean = strings.Map(func(r rune) rune {
		if r == '\x00' || r < 0x20 {
			return -1
		}
		return r
	}, clean)
	clean = leadingDotsRe.ReplaceAllString(clean, "")

	ext := strings.ToLower(filepath.Ext(clean))
	base := strings.TrimSuffix(clean, ext)

	if ext == "" {
		if exts, _ := mime.ExtensionsByType(uf.MimeType); len(exts) > 0 {
			ext = exts[0]
		}
	}

	base = fileSafeRe.ReplaceAllString(base, "-")
	base = strings.Trim(base, "- .")

	if len(base) > maxBaseNameLen {
		base = base[:maxBaseNameLen]
	}

	if base == "" {
		base = "file"
	}

	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if ext == "" {
		ext = ".bin"
	}

	safeFileName := base + ext

	now := time.Now().UTC()
	return fmt.Sprintf(
		"documents/%04d/%02d/%02d/%s/%s/%s",
		now.Year(), int(now.Month()), now.Day(),
		now.Format("20060102T150405.000000000Z"),
		strings.ToLower(strings.ReplaceAll(userUUID.String(), "-", "")),
		safeFileName,
	)
}

func (ufs *UserFileService) DeleteUserFiles(
	ctx context.Context,
	userUUID user.UUID,
) error {
	id, err := ufs.userRepository.FetchInternalID(ctx, userUUID)
	if err != nil {
		return err
	}

	// example: delete objs from s3
	// ufs.s3.DeleteObjects(ufs.userFileRepository.FetchUserFiles(...))

	if err = ufs.userFileRepository.DeleteUserFiles(ctx, id); err != nil {
		return err
	}

	return nil
}

// sanitizeFileName make file name ASCII standard
func sanitizeFileName(original string) string {
	if original == "" {
		return "file"
	}

	s := strings.TrimSpace(original)
	s = strings.ReplaceAll(s, "\\", "/")
	s = path.Base(s)

	if s == "." || s == ".." || s == "" {
		return "file"
	}

	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	s, _, _ = transform.String(t, s)

	ext := strings.ToLower(path.Ext(s))
	base := strings.TrimSuffix(s, ext)

	//  [a-z0-9], '-' и '_', dot/space → '-'
	var b strings.Builder
	b.Grow(len(base))
	prevDash := false
	for _, r := range base {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			prevDash = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		case r == '-' || r == '_':
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
		case r == '.' || unicode.IsSpace(r):
			if !prevDash {
				b.WriteRune('-')
				prevDash = true
			}
		default:
		}
	}
	base = strings.Trim(b.String(), "-")

	if base == "" {
		base = "file"
	}
	if _, bad := windowsReserved[base]; bad {
		base = "_" + base
	}

	for utf8.RuneCountInString(base)+len(ext) > maxBaseNameLen {
		_, size := utf8.DecodeLastRuneInString(base)
		if size <= 0 || size > len(base) {
			break
		}
		base = base[:len(base)-size]
	}

	return base + ext
}

func isMn(r rune) bool { return unicode.Is(unicode.Mn, r) }
