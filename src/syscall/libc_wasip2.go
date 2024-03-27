//go:build wasip2

// mini libc wrapping wasi preview2 calls in a libc api

package syscall

import (
	"strings"
	"unsafe"

	"github.com/ydnar/wasm-tools-go/cm"
	"github.com/ydnar/wasm-tools-go/wasi/cli/environment"
	"github.com/ydnar/wasm-tools-go/wasi/cli/stderr"
	"github.com/ydnar/wasm-tools-go/wasi/cli/stdin"
	"github.com/ydnar/wasm-tools-go/wasi/cli/stdout"
	wallclock "github.com/ydnar/wasm-tools-go/wasi/clocks/wall-clock"
	"github.com/ydnar/wasm-tools-go/wasi/filesystem/preopens"
	"github.com/ydnar/wasm-tools-go/wasi/filesystem/types"
	ioerror "github.com/ydnar/wasm-tools-go/wasi/io/error"
	"github.com/ydnar/wasm-tools-go/wasi/io/streams"
	"github.com/ydnar/wasm-tools-go/wasi/random/random"
)

func goString(cstr *byte) string {
	return unsafe.String(cstr, strlen(cstr))
}

//go:export strlen
func strlen(cstr *byte) uintptr {
	if cstr == nil {
		return 0
	}
	ptr := unsafe.Pointer(cstr)
	var i uintptr
	for p := (*byte)(ptr); *p != 0; p = (*byte)(unsafe.Add(unsafe.Pointer(p), 1)) {
		i++
	}
	return i
}

// ssize_t write(int fd, const void *buf, size_t count)
//
//go:export write
func write(fd int32, buf *byte, count uint) int {
	if stream, ok := wasiStreams[fd]; ok {
		return writeStream(stream, buf, count, 0)
	}

	stream, ok := wasiFiles[fd]
	if !ok {
		libcErrno = EBADF
		return -1
	}
	if stream.d == cm.ResourceNone {
		libcErrno = EBADF
		return -1
	}

	n := pwrite(fd, buf, count, int64(stream.offset))
	if n == -1 {
		return -1
	}
	stream.offset += int64(n)
	return int(n)
}

// ssize_t read(int fd, void *buf, size_t count);
//
//go:export read
func read(fd int32, buf *byte, count uint) int {
	if stream, ok := wasiStreams[fd]; ok {
		return readStream(stream, buf, count, 0)
	}

	stream, ok := wasiFiles[fd]
	if !ok {
		libcErrno = EBADF
		return -1
	}
	if stream.d == cm.ResourceNone {
		libcErrno = EBADF
		return -1
	}

	n := pread(fd, buf, count, int64(stream.offset))
	if n == -1 {
		// error during pread
		return -1
	}
	stream.offset += int64(n)
	return int(n)
}

// At the moment, each time we have a file read or write we create a new stream.  Future implementations
// could change the current in or out file stream lazily.  We could do this by tracking input and output
// offsets individually, and if they don't match the current main offset, reopen the file stream at that location.

type wasiFile struct {
	d      types.Descriptor
	oflag  int32 // orignal open flags: O_RDONLY, O_WRONLY, O_RDWR
	offset int64 // current fd offset; updated with each read/write
}

// Need to figure out which system calls we're using:
//   stdin/stdout/stderr want streams, so we use stream read/write
//   but for regular files we can use the descriptor and explicitly write a buffer to the offset?
//   The mismatch comes from trying to combine these.

var wasiFiles map[int32]*wasiFile = make(map[int32]*wasiFile)
var nextLibcFd = int32(Stderr) + 1

func findFreeFD() int32 {
	var newfd int32
	for wasiStreams[newfd] != nil || wasiFiles[newfd] != nil {
		newfd++
	}
	return newfd
}

var wasiErrno ioerror.Error

type wasiStream struct {
	in  *streams.InputStream
	out *streams.OutputStream
}

// This holds entries for stdin/stdout/stderr.

var wasiStreams map[int32]*wasiStream

func init() {
	sin := stdin.GetStdin()
	sout := stdout.GetStdout()
	serr := stderr.GetStderr()
	wasiStreams = map[int32]*wasiStream{
		0: &wasiStream{
			in: &sin,
		},
		1: &wasiStream{
			out: &sout,
		},
		2: &wasiStream{
			out: &serr,
		},
	}
}

func readStream(stream *wasiStream, buf *byte, count uint, offset int64) int {
	if stream.in == nil {
		// not a stream we can read from
		libcErrno = EBADF
		return -1
	}

	if offset != 0 {
		libcErrno = EINVAL
		return -1
	}

	libcErrno = 0
	result := stream.in.BlockingRead(uint64(count))
	if err := result.Err(); err != nil {
		if err.Closed() {
			libcErrno = 0
			return 0
		} else if err := err.LastOperationFailed(); err != nil {
			wasiErrno = *err
			libcErrno = EWASIERROR
		}
		return -1
	}

	dst := unsafe.Slice(buf, count)
	list := result.OK()
	copy(dst, list.Slice())
	return int(list.Len())
}

func writeStream(stream *wasiStream, buf *byte, count uint, offset int64) int {
	if stream.out == nil {
		// not a stream we can write to
		libcErrno = EBADF
		return -1
	}

	if offset != 0 {
		libcErrno = EINVAL
		return -1
	}

	src := unsafe.Slice(buf, count)
	var remaining = count

	// The blocking-write-and-flush call allows a maximum of 4096 bytes at a time.
	// We loop here by instead of doing subscribe/check-write/poll-one/write by hand.
	for remaining > 0 {
		len := uint(4096)
		if len > remaining {
			len = remaining
		}
		result := stream.out.BlockingWriteAndFlush(cm.ToList(src[:len]))
		if err := result.Err(); err != nil {
			if err.Closed() {
				libcErrno = 0
				return 0
			} else if err := err.LastOperationFailed(); err != nil {
				wasiErrno = *err
				libcErrno = EWASIERROR
			}
			return -1
		}
		remaining -= len
	}

	return int(count)
}

//go:linkname memcpy runtime.memcpy
func memcpy(dst, src unsafe.Pointer, size uintptr)

// ssize_t pread(int fd, void *buf, size_t count, off_t offset);
//
//go:export pread
func pread(fd int32, buf *byte, count uint, offset int64) int {
	// TODO(dgryski): Need to be consistent about all these checks; EBADF/EINVAL/... ?

	if stream, ok := wasiStreams[fd]; ok {
		return readStream(stream, buf, count, offset)

	}

	streams, ok := wasiFiles[fd]
	if !ok {
		// TODO(dgryski): EINVAL?
		libcErrno = EBADF
		return -1
	}
	if streams.d == cm.ResourceNone {
		libcErrno = EBADF
		return -1
	}
	if streams.oflag&O_RDONLY == 0 {
		libcErrno = EBADF
		return -1
	}

	result := streams.d.Read(types.FileSize(count), types.FileSize(offset))
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	list := result.OK().F0
	copy(unsafe.Slice(buf, count), list.Slice())

	// TODO(dgryski): EOF bool is ignored?
	return int(list.Len())
}

// ssize_t pwrite(int fd, void *buf, size_t count, off_t offset);
//
//go:export pwrite
func pwrite(fd int32, buf *byte, count uint, offset int64) int {
	// TODO(dgryski): Need to be consistent about all these checks; EBADF/EINVAL/... ?
	if stream, ok := wasiStreams[fd]; ok {
		return writeStream(stream, buf, count, 0)
	}

	streams, ok := wasiFiles[fd]
	if !ok {
		// TODO(dgryski): EINVAL?
		libcErrno = EBADF
		return -1
	}
	if streams.d == cm.ResourceNone {
		libcErrno = EBADF
		return -1
	}
	if streams.oflag&O_WRONLY == 0 {
		libcErrno = EBADF
		return -1
	}

	result := streams.d.Write(cm.NewList(buf, count), types.FileSize(offset))
	if err := result.Err(); err != nil {
		// TODO(dgryski):
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	return int(*result.OK())
}

// ssize_t lseek(int fd, off_t offset, int whence);
//
//go:export lseek
func lseek(fd int32, offset int64, whence int) int64 {
	if _, ok := wasiStreams[fd]; ok {
		// can't lseek a stream
		libcErrno = EBADF
		return -1
	}

	stream, ok := wasiFiles[fd]
	if !ok {
		libcErrno = EBADF
		return -1
	}
	if stream.d == cm.ResourceNone {
		libcErrno = EBADF
		return -1
	}

	switch whence {
	case 0: // SEEK_SET
		stream.offset = offset
	case 1: // SEEK_CUR
		stream.offset += offset
	case 2: // SEEK_END
		result := stream.d.Stat()
		if err := result.Err(); err != nil {
			libcErrno = errorCodeToErrno(*err)
			return -1
		}
		stream.offset = int64(result.OK().Size) + offset
	}

	return int64(stream.offset)
}

// int close(int fd)
//
//go:export close
func close(fd int32) int32 {
	if _, ok := wasiStreams[fd]; ok {
		// TODO(dgryski): Do we need to do any stdin/stdout/stderr cleanup here?
		delete(wasiStreams, fd)
		return 0
	}

	streams, ok := wasiFiles[fd]
	if !ok {
		libcErrno = EBADF
		return -1
	}
	if streams.d != cm.ResourceNone {
		streams.d.ResourceDrop()
		streams.d = 0
	}
	delete(wasiFiles, fd)

	return 0
}

// int dup(int fd)
//
//go:export dup
func dup(fd int32) int32 {
	// is fd a stream?
	if stream, ok := wasiStreams[fd]; ok {
		newfd := findFreeFD()
		wasiStreams[newfd] = stream
		return newfd
	}

	// is fd a file?
	if file, ok := wasiFiles[fd]; ok {
		// scan for first free file descriptor
		newfd := findFreeFD()
		wasiFiles[newfd] = file
		return newfd
	}

	// unknown file descriptor
	libcErrno = EBADF
	return -1
}

// void *mmap(void *addr, size_t length, int prot, int flags, int fd, off_t offset);
//
//go:export mmap
func mmap(addr unsafe.Pointer, length uintptr, prot, flags, fd int32, offset uintptr) unsafe.Pointer {
	libcErrno = ENOSYS
	return unsafe.Pointer(^uintptr(0))
}

// int munmap(void *addr, size_t length);
//
//go:export munmap
func munmap(addr unsafe.Pointer, length uintptr) int32 {
	libcErrno = ENOSYS
	return -1
}

// int mprotect(void *addr, size_t len, int prot);
//
//go:export mprotect
func mprotect(addr unsafe.Pointer, len uintptr, prot int32) int32 {
	libcErrno = ENOSYS
	return -1
}

// int chmod(const char *pathname, mode_t mode);
//
//go:export chmod
func chmod(pathname *byte, mode uint32) int32 {
	return 0
}

// int mkdir(const char *pathname, mode_t mode);
//
//go:export mkdir
func mkdir(pathname *byte, mode uint32) int32 {
	path := goString(pathname)
	dir, relPath := findPreopenForPath(path)

	result := dir.CreateDirectoryAt(relPath)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	return 0
}

// int rmdir(const char *pathname);
//
//go:export rmdir
func rmdir(pathname *byte) int32 {
	path := goString(pathname)
	dir, relPath := findPreopenForPath(path)

	result := dir.RemoveDirectoryAt(relPath)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	return 0
}

// int rename(const char *from, *to);
//
//go:export rename
func rename(from, to *byte) int32 {
	fromPath := goString(from)
	fromDir, fromRelPath := findPreopenForPath(fromPath)

	toPath := goString(to)
	toDir, toRelPath := findPreopenForPath(toPath)

	result := fromDir.RenameAt(fromRelPath, toDir, toRelPath)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	return 0
}

// int symlink(const char *from, *to);
//
//go:export symlink
func symlink(from, to *byte) int32 {
	fromPath := goString(from)
	fromDir, fromRelPath := findPreopenForPath(fromPath)

	toPath := goString(to)
	toDir, toRelPath := findPreopenForPath(toPath)

	if fromDir != toDir {
		libcErrno = EACCES
		return -1
	}

	// TODO(dgryski): check fromDir == toDir?

	result := fromDir.SymlinkAt(fromRelPath, toRelPath)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	return 0
}

// int fsync(int fd);
//
//go:export fsync
func fsync(fd int32) int32 {
	if _, ok := wasiStreams[fd]; ok {
		// can't sync a stream
		libcErrno = EBADF
		return -1
	}

	streams, ok := wasiFiles[fd]
	if !ok {
		libcErrno = EBADF
		return -1
	}
	if streams.d == cm.ResourceNone {
		libcErrno = EBADF
		return -1
	}
	if streams.oflag&O_WRONLY == 0 {
		libcErrno = EBADF
		return -1
	}

	result := streams.d.SyncData()
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	return 0
}

// ssize_t readlink(const char *path, void *buf, size_t count);
//
//go:export readlink
func readlink(pathname *byte, buf *byte, count uint) int {
	path := goString(pathname)
	dir, relPath := findPreopenForPath(path)

	result := dir.ReadLinkAt(relPath)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	s := *result.OK()
	size := uintptr(count)
	if size > uintptr(len(s)) {
		size = uintptr(len(s))
	}

	memcpy(unsafe.Pointer(buf), unsafe.Pointer(unsafe.StringData(s)), size)
	return int(size)
}

// int unlink(const char *pathname);
//
//go:export unlink
func unlink(pathname *byte) int32 {
	path := goString(pathname)
	dir, relPath := findPreopenForPath(path)

	result := dir.UnlinkFileAt(relPath)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	return 0
}

// int getpagesize(void);
//
//go:export getpagesize
func getpagesize() int {
	return 0

}

// int stat(const char *path, struct stat * buf);
//
//go:export stat
func stat(pathname *byte, dst *Stat_t) int32 {
	path := goString(pathname)
	dir, relPath := findPreopenForPath(path)

	result := dir.StatAt(0, relPath)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	setStatFromWASIStat(dst, result.OK())

	return 0
}

// int fstat(int fd, struct stat * buf);
//
//go:export fstat
func fstat(fd int32, dst *Stat_t) int32 {
	if _, ok := wasiStreams[fd]; ok {
		// TODO(dgryski): fill in stat buffer for stdin etc
		return -1
	}

	stream, ok := wasiFiles[fd]
	if !ok {
		libcErrno = EBADF
		return -1
	}
	if stream.d == cm.ResourceNone {
		libcErrno = EBADF
		return -1
	}
	result := stream.d.Stat()
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	setStatFromWASIStat(dst, result.OK())

	return 0
}

func setStatFromWASIStat(sstat *Stat_t, wstat *types.DescriptorStat) {
	// This will cause problems for people who want to compare inodes
	sstat.Dev = 0
	sstat.Ino = 0
	sstat.Rdev = 0

	sstat.Nlink = uint64(wstat.LinkCount)

	// No mode bits
	sstat.Mode = 0

	// No uid/gid
	sstat.Uid = 0
	sstat.Gid = 0
	sstat.Size = int64(wstat.Size)

	// made up numbers
	sstat.Blksize = 512
	sstat.Blocks = (sstat.Size + 511) / int64(sstat.Blksize)

	setOptTime := func(t *Timespec, o *wallclock.DateTime) {
		t.Sec = 0
		t.Nsec = 0
		if o != nil {
			t.Sec = int32(o.Seconds)
			t.Nsec = int64(o.Nanoseconds)
		}
	}

	setOptTime(&sstat.Atim, wstat.DataAccessTimestamp.Some())
	setOptTime(&sstat.Mtim, wstat.DataModificationTimestamp.Some())
	setOptTime(&sstat.Ctim, wstat.StatusChangeTimestamp.Some())
}

// int lstat(const char *path, struct stat * buf);
//
//go:export lstat
func lstat(pathname *byte, dst *Stat_t) int32 {
	path := goString(pathname)
	dir, relPath := findPreopenForPath(path)

	result := dir.StatAt(0, relPath)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	setStatFromWASIStat(dst, result.OK())

	return 0
}

func init() {
	populatePreopens()
}

var wasiCWD types.Descriptor

var wasiPreopens map[string]types.Descriptor

func populatePreopens() {
	dirs := preopens.GetDirectories().Slice()
	preopens := make(map[string]types.Descriptor, len(dirs))
	for _, tup := range dirs {
		desc, path := tup.F0, tup.F1
		preopens[path] = desc
		if path == "." {
			wasiCWD = desc
		}
	}
	wasiPreopens = preopens
}

// FIXME(ydnar): opening a stripped path fails, so ignore it.
func findPreopenForPath(path string) (types.Descriptor, string) {
	if strings.HasPrefix(path, "./") || path == "." {
		return wasiCWD, path
	}

	for k, v := range wasiPreopens {
		if strings.HasPrefix(path, k) {
			if path == k {
				path = "."
			} else {
				path = strings.TrimPrefix(path, k+"/")
			}
			return v, path
		}
	}
	return wasiCWD, path
}

// int open(const char *pathname, int flags, mode_t mode);
//
//go:export open
func open(pathname *byte, flags int32, mode uint32) int32 {
	path := goString(pathname)
	dir, relPath := findPreopenForPath(path)

	var dflags types.DescriptorFlags
	if (flags & O_RDONLY) == O_RDONLY {
		dflags |= types.DescriptorFlagsRead
	}
	if (flags & O_WRONLY) == O_WRONLY {
		dflags |= types.DescriptorFlagsWrite
	}

	var oflags types.OpenFlags
	if flags&O_CREAT == O_CREAT {
		oflags |= types.OpenFlagsCreate
	}
	if flags&O_DIRECTORY == O_DIRECTORY {
		oflags |= types.OpenFlagsDirectory
	}
	if flags&O_EXCL == O_EXCL {
		oflags |= types.OpenFlagsExclusive
	}
	if flags&O_TRUNC == O_TRUNC {
		oflags |= types.OpenFlagsTruncate
	}

	// By default, follow symlinks for open() unless O_NOFOLLOW was passed
	var pflags types.PathFlags = types.PathFlagsSymlinkFollow
	if flags&O_NOFOLLOW == O_NOFOLLOW {
		// O_NOFOLLOW was passed, so turn off SymlinkFollow
		pflags &^= types.PathFlagsSymlinkFollow
	}

	result := dir.OpenAt(pflags, relPath, oflags, dflags)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	stream := wasiFile{
		d:     *result.OK(),
		oflag: flags,
	}

	if flags&(O_WRONLY|O_APPEND) == (O_WRONLY | O_APPEND) {
		result := stream.d.Stat()
		if err := result.Err(); err != nil {
			libcErrno = errorCodeToErrno(*err)
			return -1
		}
		stream.offset = int64(result.OK().Size)
	}

	libcfd := nextLibcFd
	nextLibcFd++

	wasiFiles[libcfd] = &stream

	return int32(libcfd)
}

func errorCodeToErrno(err types.ErrorCode) Errno {
	switch err {
	case types.ErrorCodeAccess:
		return EACCES
	case types.ErrorCodeWouldBlock:
		return EAGAIN
	case types.ErrorCodeAlready:
		return EALREADY
	case types.ErrorCodeBadDescriptor:
		return EBADF
	case types.ErrorCodeBusy:
		return EBUSY
	case types.ErrorCodeDeadlock:
		return EDEADLK
	case types.ErrorCodeQuota:
		return EDQUOT
	case types.ErrorCodeExist:
		return EEXIST
	case types.ErrorCodeFileTooLarge:
		return EFBIG
	case types.ErrorCodeIllegalByteSequence:
		return EILSEQ
	case types.ErrorCodeInProgress:
		return EINPROGRESS
	case types.ErrorCodeInterrupted:
		return EINTR
	case types.ErrorCodeInvalid:
		return EINVAL
	case types.ErrorCodeIO:
		return EIO
	case types.ErrorCodeIsDirectory:
		return EISDIR
	case types.ErrorCodeLoop:
		return ELOOP
	case types.ErrorCodeTooManyLinks:
		return EMLINK
	case types.ErrorCodeMessageSize:
		return EMSGSIZE
	case types.ErrorCodeNameTooLong:
		return ENAMETOOLONG
	case types.ErrorCodeNoDevice:
		return ENODEV
	case types.ErrorCodeNoEntry:
		return ENOENT
	case types.ErrorCodeNoLock:
		return ENOLCK
	case types.ErrorCodeInsufficientMemory:
		return ENOMEM
	case types.ErrorCodeInsufficientSpace:
		return ENOSPC
	case types.ErrorCodeNotDirectory:
		return ENOTDIR
	case types.ErrorCodeNotEmpty:
		return ENOTEMPTY
	case types.ErrorCodeNotRecoverable:
		return ENOTRECOVERABLE
	case types.ErrorCodeUnsupported:
		return ENOSYS
	case types.ErrorCodeNoTTY:
		return ENOTTY
	case types.ErrorCodeNoSuchDevice:
		return ENXIO
	case types.ErrorCodeOverflow:
		return EOVERFLOW
	case types.ErrorCodeNotPermitted:
		return EPERM
	case types.ErrorCodePipe:
		return EPIPE
	case types.ErrorCodeReadOnly:
		return EROFS
	case types.ErrorCodeInvalidSeek:
		return ESPIPE
	case types.ErrorCodeTextFileBusy:
		return ETXTBSY
	case types.ErrorCodeCrossDevice:
		return EXDEV
	}
	return Errno(err)
}

type libc_DIR struct {
	d types.DirectoryEntryStream
}

// DIR *fdopendir(int);
//
//go:export fdopendir
func fdopendir(fd int32) unsafe.Pointer {
	if _, ok := wasiStreams[fd]; ok {
		libcErrno = EBADF
		return nil
	}

	stream, ok := wasiFiles[fd]
	if !ok {
		libcErrno = EBADF
		return nil
	}
	if stream.d == cm.ResourceNone {
		libcErrno = EBADF
		return nil
	}

	result := stream.d.ReadDirectory()
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return nil
	}

	return unsafe.Pointer(&libc_DIR{d: *result.OK()})
}

// int fdclosedir(DIR *);
//
//go:export fdclosedir
func fdclosedir(dirp unsafe.Pointer) int32 {
	if dirp == nil {
		return 0

	}
	dir := (*libc_DIR)(dirp)
	if dir.d == cm.ResourceNone {
		return 0
	}

	dir.d.ResourceDrop()
	dir.d = cm.ResourceNone

	return 0
}

// struct dirent *readdir(DIR *);
//
//go:export readdir
func readdir(dirp unsafe.Pointer) *Dirent {
	if dirp == nil {
		return nil

	}
	dir := (*libc_DIR)(dirp)
	if dir.d == cm.ResourceNone {
		return nil
	}

	result := dir.d.ReadDirectoryEntry()
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return nil
	}

	entry := result.OK().Some()
	if entry == nil {
		libcErrno = 0
		return nil
	}

	// The dirent C struct uses a flexible array member to indicate that the
	// directory name is laid out in memory right after the struct data:
	//
	// struct dirent {
	//   ino_t d_ino;
	//   unsigned char d_type;
	//   char d_name[];
	// };
	buf := make([]byte, unsafe.Sizeof(Dirent{})+uintptr(len(entry.Name)))
	dirent := (*Dirent)((unsafe.Pointer)(&buf[0]))

	// No inodes in wasi
	dirent.Ino = 0
	dirent.Type = p2fileTypeToLibcType(entry.Type)
	copy(buf[unsafe.Offsetof(dirent.Type)+1:], entry.Name)

	return dirent
}

func p2fileTypeToLibcType(t types.DescriptorType) uint8 {
	switch t {
	case types.DescriptorTypeUnknown:
		return DT_UNKNOWN
	case types.DescriptorTypeBlockDevice:
		return DT_BLK
	case types.DescriptorTypeCharacterDevice:
		return DT_CHR
	case types.DescriptorTypeDirectory:
		return DT_DIR
	case types.DescriptorTypeFIFO:
		return DT_FIFO
	case types.DescriptorTypeSymbolicLink:
		return DT_LNK
	case types.DescriptorTypeRegularFile:
		return DT_REG
	case types.DescriptorTypeSocket:
		return DT_FIFO
	}

	return DT_UNKNOWN
}

var libc_envs map[string]string

func init() {
	libc_envs = make(map[string]string)
	for _, kv := range environment.GetEnvironment().Slice() {
		libc_envs[kv[0]] = kv[1]
	}
}

// char * getenv(const char *name);
//
//go:export getenv
func getenv(key *byte) *byte {
	k := goString(key)

	v, ok := libc_envs[k]
	if !ok {
		return nil
	}

	// The new allocation is zero-filled; allocating an extra byte and then
	// copying the data over will leave the last byte untouched,
	// null-terminating the string.
	vbytes := make([]byte, len(v)+1)
	copy(vbytes, v)
	return unsafe.SliceData(vbytes)
}

// int setenv(const char *name, const char *value, int overwrite);
//
//go:export setenv
func setenv(key, value *byte, overwrite int) int {
	k := goString(key)
	if _, ok := libc_envs[k]; ok && overwrite == 0 {
		return 0
	}

	v := goString(value)
	libc_envs[k] = v

	return 0
}

// int unsetenv(const char *name);
//
//go:export unsetenv
func unsetenv(key *byte) int {
	k := goString(key)
	delete(libc_envs, k)
	return 0
}

// void arc4random_buf (void *, size_t);
//
//go:export arc4random_buf
func arc4random_buf(p unsafe.Pointer, l uint) {
	result := random.GetRandomBytes(uint64(l))
	s := result.Slice()
	memcpy(unsafe.Pointer(p), unsafe.Pointer(unsafe.SliceData(s)), uintptr(l))
}

var libc_cwd string

// int chdir(char *name)
//
//go:export chdir
func chdir(name *byte) int {
	path := goString(name)
	dir, rel := findPreopenForPath(path)

	result := dir.OpenAt(types.PathFlagsSymlinkFollow, rel, types.OpenFlagsDirectory, types.DescriptorFlagsRead)
	if err := result.Err(); err != nil {
		libcErrno = errorCodeToErrno(*err)
		return -1
	}

	libc_cwd = path
	wasiCWD = *result.OK()
	return 0
}

// char *getcwd(char *buf, size_t size)
//
//go:export getcwd
func getcwd(buf *byte, size uint) *byte {
	if size > uint(len(libc_cwd)) {
		size = uint(len(libc_cwd))
	}

	// TODO(dgryski): null termination?
	memcpy(unsafe.Pointer(buf), unsafe.Pointer(unsafe.SliceData([]byte(libc_cwd))), uintptr(size))
	return buf
}
