## share
to automate personal files's protection, something like git 用类似于git的方式，自动化保护个人文件

record file operation: **upload** **download** **modify** **duplicate** **move** **delete** **drop**

recover trace file to any history version

**usage: share** [*subcommand*] [*arguments*]

arguments has two style: indexed and named

indexed arguments have only value, different position has different meanings

named arguments have both name and value, like name=value

## tutorial
use **share trace** *filename* start to trace file

use **share fork** *filename* replace **cp** command

use **share trash** *filename* replace **rm** command

use **share move** *oldfilename* *newfilename* replace **mv** command

use **share trace** *filename* every time when the file change

use **share listen** to start a server, then open browser to trans files with other people

use **share drop** *filename* stop to trace file

use **share mark** *filename* *hash* mark file trace log

use **share show** show all trace file

use **share show** *filename* show file trace log

use **share show** *filename* *hash* show file trace same log

use **share show** *filename* *hash* *recoverfile* receover file to special version

## manual
### share help [cmd] [args] 
show share usage 

* **cmd** (=dump) sub command name

### share dump [dstfile] [args] 
dump markdown document of help

* **dstfile** destination file name

### share fork [srcfile [dstfile]] [args] 
trace file's copying

* **srcfile** source file name
* **dstfile** destination file name

### share trash [srcfile] [args] 
trace file's trash

* **srcfile** source file name

### share mark [srcfile [hash [mark]]] [args] 
mark file log

* **srcfile** source file name
* **hash** the hash value of file
* **mark** mark the file log

### share listen [addr [share [log]]] [args] 
start server to trace files' transport

* **addr** (=:9090) socket listen address
* **share** (=/home/shy/share) share dirent path
* **log** (=/home/shy/share/.trash/.share.log) trash log file

### share trace [srcfile] [args] 
trace file

* **srcfile** source file name

### share move [srcfile [dstfile]] [args] 
trace file's moving

* **srcfile** source file name
* **dstfile** destination file name

### share drop [srcfile] [args] 
stop trace file

* **srcfile** source file name

### share show [srcfile [hash [dstfile]]] [args] 
show file log

* **srcfile** source file name
* **hash** the hash value of file
* **dstfile** destination file name

## appendix
### other optional arguments
* **mark** mark the file log
* **addr** (=:9090) socket listen address
* **share** (=/home/shy/share) share dirent path
* **action** (=trace) trace file action
* **srcfile** source file name
* **log** (=/home/shy/share/.trash/.share.log) trash log file
* **remote** (=127.0.0.1:9090) socket remote addr
* **trash** (=/home/shy/share/.trash) trash dirent path
* **dbuser** (=share) database user name
* **dbword** (=share) database pass word
* **cmd** (=dump) sub command name
* **dstfile** destination file name
* **hash** the hash value of file
* **dbfile** (=/home/shy/share/.trash/.share.db) trash database file
* **dbname** (=share) database name
