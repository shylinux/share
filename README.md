## share
to automate personal files's protection, something like git 用类似于git的方式，自动化保护个人文件

usage: share [subcommand] [arguments]

arguments has two style: indexed and named

indexed arguments have only value, different position has different meanings

named arguments have both name and value, like name=value

### share trace [srcfile] [args] 
begin to trace file

* **srcfile** (=) source file name

### share fork [srcfile [dstfile]] [args] 
copy file 

* **srcfile** (=) source file name
* **dstfile** (=/Users/shaoying/go/src/share/hi.md) destination file name

### share trash [srcfile] [args] 
move file to trash

* **srcfile** (=) source file name

### share show [srcfile [hash [dstfile]]] [args] 
show file log

* **srcfile** (=) source file name
* **hash** (=) the hash value of file
* **dstfile** (=/Users/shaoying/go/src/share/hi.md) destination file name

### share mark [srcfile [hash [mark]]] [args] 
mark file log

* **srcfile** (=) source file name
* **hash** (=) the hash value of file
* **mark** (=) mark the file log

### share dump [dstfile] [args] 
dump help document

* **dstfile** (=/Users/shaoying/go/src/share/hi.md) destination file name

### share help [cmd] [args] 
show share usage help

* **cmd** (=dump) sub command name

### share listen [addr [share [log]]] [args] 
socket listen address

* **addr** (=:9090) socket listen address
* **share** (=./temp) share dirent path
* **log** (=./share.log) trash dirent path

### share move [srcfile [dstfile]] [args] 
move file 

* **srcfile** (=) source file name
* **dstfile** (=/Users/shaoying/go/src/share/hi.md) destination file name

### share drop [srcfile] [args] 
stop trace file

* **srcfile** (=) source file name

### other optional arguments

* **action** (=trace) trace file action
* **dstfile** (=/Users/shaoying/go/src/share/hi.md) destination file name
* **dbword** (=share) database pass word
* **hash** (=) the hash value of file
* **mark** (=) mark the file log
* **dbuser** (=share) database user name
* **srcfile** (=) source file name
* **addr** (=:9090) socket listen address
* **trash** (=./trash) trash dirent path
* **cmd** (=dump) sub command name
* **remote** (=127.0.0.1:9090) socket remote addr
* **share** (=./temp) share dirent path
* **log** (=./share.log) trash dirent path
* **dbname** (=share) database name
