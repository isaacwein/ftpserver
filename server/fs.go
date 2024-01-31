package server

type FtpFS interface {
	Dir()
	File()
	Open()
	Create()
	Remove()
	Rename()
	Stat()
	Lstat()
}
type FtpLocalFS struct {
	Root string
}

func (r *FtpLocalFS) Dir() {

}

func (r *FtpLocalFS) File() {

}

func (r *FtpLocalFS) Open() {

}
func (r *FtpLocalFS) Create() {

}
func (r *FtpLocalFS) Remove() {

}
func (r *FtpLocalFS) Rename() {

}
func (r *FtpLocalFS) Stat() {

}
func (r *FtpLocalFS) Lstat() {

}

func NewFtpLocalFS(root string) *FtpLocalFS {
	return &FtpLocalFS{}

}
