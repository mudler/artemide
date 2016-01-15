package archiveutils

import (
	"bufio"
	"compress/gzip"
	"io"

	"github.com/docker/docker/pkg/archive"
)

const compressionBufSize = 32768

var RsyncDefaultOpts = []string{"-av", "--delete"}

func ExtractTarGz(in io.Reader, dest string) (err error) {
	//func ExtractTarGz(in io.Reader, dest string, uid int, gid int) (err error) {

	//    ChownOpts: &archive.TarChownOptions{
	//			UID: uid,
	//			GID: gid,
	//		},

	return archive.Untar(in, dest, &archive.TarOptions{
		Compression:     archive.Gzip,
		NoLchown:        false,
		ExcludePatterns: []string{"dev/"}, // prevent operation not permitted
	})
}

func ExtractTar(in io.Reader, dest string) (err error) {
	//func ExtractTarGz(in io.Reader, dest string, uid int, gid int) (err error) {

	//    ChownOpts: &archive.TarChownOptions{
	//			UID: uid,
	//			GID: gid,
	//		},

	return archive.Untar(in, dest, &archive.TarOptions{
		NoLchown:        false,
		ExcludePatterns: []string{"dev/"}, // prevent operation not permitted
	})
}

func Compress(in io.Reader) io.ReadCloser {
	pReader, pWriter := io.Pipe()
	bufWriter := bufio.NewWriterSize(pWriter, compressionBufSize)
	compressor := gzip.NewWriter(bufWriter)

	go func() {
		_, err := io.Copy(compressor, in)
		if err == nil {
			err = compressor.Close()
		}
		if err == nil {
			err = bufWriter.Flush()
		}
		if err != nil {
			pWriter.CloseWithError(err)
		} else {
			pWriter.Close()
		}
	}()

	return pReader
}
