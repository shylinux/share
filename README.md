## share
to automate personal files's protection, something like git 用类似于git的方式，自动化保护个人文件

record file operation: **upload** **download** **modify** **duplicate** **move** **delete** **drop**

recover trace file to any history version

usage: share [*subcommand*] [*arguments*]

arguments has two style: indexed and named

indexed arguments have only value, different position has different meanings

named arguments have both name and value, like name=value

### share dump [dstfile] [args] 
dump markdown document of help

* **dstfile** destination file name

### share listen [addr [share [log]]] [args] 
start server to trace files' transport

* **addr** (=:9090) socket listen address
* **share** (=./temp) share dirent path
* **log** (=./share.log) trash dirent path

### share trash [srcfile] [args] 
trace file's trash

* **srcfile** source file name

### share drop [srcfile] [args] 
stop trace file

* **srcfile** source file name

### share mark [srcfile [hash [mark]]] [args] 
mark file log

* **srcfile** source file name
* **hash** the hash value of file
* **mark** mark the file log

### share help [cmd] [args] 
show share usage 

* **cmd** (=dump) sub command name

### share trace [srcfile] [args] 
trace file

* **srcfile** source file name

### share fork [srcfile [dstfile]] [args] 
trace file's copying

* **srcfile** source file name
* **dstfile** destination file name

### share move [srcfile [dstfile]] [args] 
trace file's moving

* **srcfile** source file name
* **dstfile** destination file name

### share show [srcfile [hash [dstfile]]] [args] 
show file log

* **srcfile** source file name
* **hash** the hash value of file
* **dstfile** destination file name

### other optional arguments

* **srcfile** source file name
* **remote** (=127.0.0.1:9090) socket remote addr
* **mark** mark the file log
* **cmd** (=dump) sub command name
* **dstfile** destination file name
* **share** (=./temp) share dirent path
* **trash** (=./trash) trash dirent path
* **action** (=trace) trace file action
* **hash** the hash value of file
* **addr** (=:9090) socket listen address
* **log** (=./share.log) trash dirent path
* **dbuser** (=share) database user name
* **dbword** (=share) database pass word
* **dbname** (=share) database name
