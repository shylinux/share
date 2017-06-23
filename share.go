package main // {{{
// }}}
import ( // {{{
	"bufio"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// }}}

type command struct { // {{{
	text string
	hand func() error
	args []string
}

// }}}
type argument struct { // {{{
	text string
	val  string
}

// }}}
var ( // {{{
	db *sql.DB
)

// }}}

func index(w http.ResponseWriter, r *http.Request) { // {{{
	var e error
	log.Printf("[%s] %s %s\n", r.RemoteAddr, r.Method, r.URL)

	if r.Method == "GET" {
		fs, e := os.Stat("." + r.URL.Path)
		if e != nil {
			log.Printf("not found")
			http.NotFound(w, r)
			return
		}

		if fs.IsDir() {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<DOCTYPE html><head><meta name='viewport' content='width=device-width, initial-scale=1.0'></head>`)
			fmt.Fprintf(w, `<body><form onsubmit='if (this.mark.value == "") {alert("must add comment"); return false} else {return true}' method='POST' action='%s' enctype='multipart/form-data'><input type='file' name='file'><br><br>请留言：<input type='text' name='mark'><input type='submit'></form>`, r.URL.Path)

			fmt.Fprintf(w, "<pre><a href='/'>home: /</a>   ")
			back := r.URL.Path[0 : len(r.URL.Path)-1]
			back = back[0 : strings.LastIndex(back, "/")+1]
			fmt.Fprintf(w, "<a href='%s'>back: %s</a>   ", back, back)
			fmt.Fprintf(w, "path: %s<hr></pre>", r.URL.Path)

			if fl, e := ioutil.ReadDir(fs.Name()); e == nil {
				fmt.Fprintf(w, "<pre>")
				for i, v := range fl {
					if v.Name()[0] == '.' {
						continue
					}

					if v.IsDir() {
						fmt.Fprintf(w, "%2d %20s    ---    <a href='%s/'>%s</a><br>", i, v.ModTime().Format("2006-01-02 15:04:05"), v.Name(), v.Name())
					} else {
						size := ""
						switch {
						case v.Size() > 10000000000:
							size = fmt.Sprintf("%dG", v.Size()/1000000000)
						case v.Size() > 10000000:
							size = fmt.Sprintf("%dM", v.Size()/1000000)
						case v.Size() > 10000:
							size = fmt.Sprintf("%dK", v.Size()/1000)
						default:
							size = fmt.Sprintf("%dB", v.Size())
						}

						fmt.Fprintf(w, "%2d %20s %6s    <a href='%s'>%s</a><br>", i, v.ModTime().Format("2006-01-02 15:04:05"), size, v.Name(), v.Name())
					}
				}
				fmt.Fprintf(w, "</pre>")
				fmt.Fprintf(w, "</body>")
				return
			}
		} else {
			if f, e := os.Open(arg("srcfile", "."+r.URL.Path)); e == nil {
				defer f.Close()

				io.Copy(w, f)

				arg("action", "GET")
				arg("mark", r.RemoteAddr)
				trace()
				return
			}
		}
	} else if r.Method == "POST" {
		if u, h, e := r.FormFile("file"); e == nil {
			defer u.Close()

			name := arg("srcfile", "."+r.URL.Path+h.Filename)

			if info, e := os.Stat(name); e == nil {
				log.Printf("%s already exists\n", info.Name())
				fmt.Fprintf(w, "%s already exists\n", info.Name())
				return
			}

			if f, e := os.Create(name); e == nil {
				defer f.Close()

				u.Seek(0, os.SEEK_SET)
				io.Copy(f, u)
				fmt.Fprintf(w, "%s upload success\n", name)

				arg("action", "POST")
				arg("mark", r.RemoteAddr+" "+r.FormValue("mark"))
				trace()
			}
		}
	}

	log.Printf("%s\n", e)
	fmt.Fprintf(w, "%s\n", e)
}

// }}}
func listen() (e error) { // {{{

	if l, e := os.OpenFile(arg("log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600); e == nil {
		log.SetOutput(l)
	}

	for k, v := range args {
		log.Printf("\t%s=%s\t%s\n", k, v.val, v.text)
	}

	os.Chdir(arg("share"))

	http.HandleFunc("/", index)

	return http.ListenAndServe(arg("addr"), nil)
}

// }}}

func filemd(file string) (md string, size int64, e error) { // {{{

	var f, fm *os.File

	if f, e = os.Open(file); e == nil {
		defer f.Close()

		h := md5.New()
		if size, e = io.Copy(h, f); e == nil {
			md = hex.EncodeToString(h.Sum(nil))
			os.MkdirAll(path.Join(arg("trash"), md[0:2]), 0700)

			if fm, e = os.Create(path.Join(arg("trash"), md[0:2], md)); e == nil {
				defer fm.Close()

				f.Seek(0, os.SEEK_SET)
				size, e = io.Copy(fm, f)

				return
			}
		}
	}

	return "", 0, e
}

// }}}
func trace() (e error) { // {{{
	fp := arg("srcfile")
	if fp == "" {
		return errors.New("srcfile name invalidate")
	}

	md, size, e := filemd(fp)
	if md == "" || size == 0 || e != nil {
		return errors.New(fmt.Sprintf("filemd error: %s", e))
	}

	action := arg("action")
	if action == "" {
		action = arg("action", "trace")
	}

	var rows *sql.Rows
	sql := "select list from name where name=?"
	if rows, e = db.Query(sql, fp); e == nil {
		if rows.Next() {
			var list string
			rows.Scan(&list)
			rows.Close()

			sql = fmt.Sprintf("select done, name, hash from %s order by time desc limit 1", list)
			if rows, e = db.Query(sql); e == nil && rows.Next() {
				var done, name, hash string
				rows.Scan(&done, &name, &hash)
				rows.Close()

				if action != done || name != fp || md != hash {
					db.Exec(fmt.Sprintf("insert into %s values(?, ?, ?, ?, ?)", list), time.Now().Unix(), action, fp, md, arg("mark"))

					db.Exec(fmt.Sprintf("insert into hash values(%d, '%s', %d, 0)", time.Now().Unix(), md, size))
					db.Exec(fmt.Sprintf("update hash set count=count+1 where hash = ?"), md)

					log.Printf("[%s] %s %s %s", arg("mark"), action, fp, md)
				}

				return nil
			}
		} else {
			rows.Close()

			count := 0
			sql = "select value from config where name='count'"
			if rows, e = db.Query(sql); e == nil && rows.Next() {
				rows.Scan(&count)
				rows.Close()

				db.Exec("update config set value=value+1 where name='count'")

				db.Exec(fmt.Sprintf("insert into name values(%d, '%s', 'file%d')", time.Now().Unix(), fp, count))

				db.Exec(fmt.Sprintf("create table if not exists file%d(time int, done char(8), name text, hash char(32), mark text)", count))
				db.Exec(fmt.Sprintf("insert into file%d values(?, ?, ?, ?, ?)", count), time.Now().Unix(), action, fp, md, arg("mark"))

				db.Exec(fmt.Sprintf("insert into hash values(%d, '%s', %d, 0)", time.Now().Unix(), md, size))
				db.Exec(fmt.Sprintf("update hash set count=count+1 where hash = ?"), md)

				log.Printf("[%s] %s %s %s", arg("mark"), action, fp, md)
				return nil
			}
		}
	}
	return
}

// }}}
func drop() (e error) { // {{{
	fp := arg("srcfile")

	var rows *sql.Rows

	if fp == "" {
		if rows, e = db.Query(fmt.Sprintf("select * from name")); e == nil {
			var t int64
			var name, list string
			var names = make([]string, 0)

			for i := 0; rows.Next(); i++ {
				rows.Scan(&t, &name, &list)
				names = append(names, name)
				fmt.Printf("%d %s %s %s\n", i, time.Unix(t, 0).Format("2006/01/02 15:04:05"), name, list)
			}
			rows.Close()

			for len(names) > 0 {
				i := -1
				fmt.Printf("select which to drop: ")
				fmt.Scanf("%d", &i)

				if i < 0 {
					break
				}

				if i < len(names) && names[i] != "" {
					arg("srcfile", names[i])
					if e = drop(); e != nil {
						break
					}

					names[i] = ""
				}
			}
		}
	} else {

		if rows, e = db.Query(fmt.Sprintf("select list from name where name = ?"), fp); e == nil && rows.Next() {
			var list string
			rows.Scan(&list)
			rows.Close()

			if rows, e = db.Query(fmt.Sprintf("select hash from %s", list)); e == nil {

				var hashs = make([]string, 0)
				var hash string

				for rows.Next() {
					rows.Scan(&hash)
					hashs = append(hashs, hash)
				}
				rows.Close()

				for _, hash = range hashs {
					db.Exec(fmt.Sprintf("update hash set count=count-1 where hash=?"), hash)
				}

				db.Exec(fmt.Sprintf("drop table %s", list))
				db.Exec(fmt.Sprintf("delete from name where name = ?"), fp)

				if rows, e = db.Query(fmt.Sprintf("select hash from hash where count=0")); e == nil {
					for rows.Next() {
						rows.Scan(&hash)
						os.Remove(path.Join(arg("trash"), hash[0:2], hash))
						os.Remove(path.Join(arg("trash"), hash[0:2]))
					}
					rows.Close()

					db.Exec(fmt.Sprintf("delete from hash where count = 0"))
				}

				log.Printf("[%s] drop %s", arg("mark"), fp)
			}
		}
	}
	return
}

// }}}

func show() (e error) { // {{{
	var rows *sql.Rows

	if arg("srcfile") == "" {
		if rows, e = db.Query(fmt.Sprintf("select * from name")); e == nil {
			var t int64
			var name, list string
			var names = make([]string, 0)

			for rows.Next() {
				rows.Scan(&t, &name, &list)
				names = append(names, name)
				fmt.Printf("%d %s %s %s\n", len(names)-1, time.Unix(t, 0).Format("2006/01/02 15:04:05"), name, list)
			}
			rows.Close()

			for len(names) > 0 {
				i := -1
				fmt.Printf("select which to show: ")
				fmt.Scanf("%d", &i)

				if i < 0 {
					break
				}

				if i < len(names) {
					arg("srcfile", names[i])
					show()
				}
			}
		}
	} else {
		if rows, e = db.Query(fmt.Sprintf("select list from name where name = ?"), arg("srcfile")); e == nil && rows.Next() {
			var list string
			rows.Scan(&list)
			rows.Close()

			if rows, e = db.Query(fmt.Sprintf("select * from %s where hash like '%s%%'", list, arg("hash"))); e == nil {
				var i, t int64
				var done, name, hash, user string

				for i = 0; rows.Next(); i++ {
					rows.Scan(&t, &done, &name, &hash, &user)

					fmt.Printf("%s %s %s %s %s\n", time.Unix(t, 0).Format("2006/01/02 15:04:05"), done, name, hash, user)
				}

				rows.Close()

				if arg("dstfile") != "" {
					var f, fm *os.File
					if f, e = os.Open(path.Join(arg("trash"), hash[0:2], hash)); e == nil {
						defer f.Close()

						if fm, e = os.Create(arg("dstfile")); e == nil {
							defer fm.Close()
							io.Copy(fm, f)
						}
					}
				}
			}
		}
	}
	return
}

// }}}
func mark() (e error) { // {{{
	var rows *sql.Rows

	fp := arg("srcfile")
	if fp == "" {
		if rows, e = db.Query(fmt.Sprintf("select * from name")); e == nil {
			var t int64
			var name, list string
			var names = make([]string, 0)

			for rows.Next() {
				rows.Scan(&t, &name, &list)
				names = append(names, name)
				fmt.Printf("%d %s %s %s\n", len(names)-1, time.Unix(t, 0).Format("2006/01/02 15:04:05"), name, list)
			}

			rows.Close()

			for len(names) > 0 {
				i := -1
				fmt.Printf("select which file to mark: ")
				fmt.Scanf("%d", &i)

				if i < 0 {
					break
				}

				fp = arg("srcfile", names[i])
				mark()
			}
		}
	} else {

		if rows, e = db.Query(fmt.Sprintf("select list from name where name = ?"), fp); e == nil && rows.Next() {
			var list string
			rows.Scan(&list)
			rows.Close()

			if rows, e := db.Query(fmt.Sprintf("select * from %s", list)); e == nil {
				var t int64
				var done, name, hash, user string
				var names = make([]string, 0)
				var hashs = make([]string, 0)
				var times = make([]int64, 0)

				for i := 0; rows.Next(); i++ {
					rows.Scan(&t, &done, &name, &hash, &user)
					names = append(names, name)
					hashs = append(hashs, hash)
					times = append(times, t)

					fmt.Printf("%d %s %s %s %s %s\n", len(hashs)-1, time.Unix(t, 0).Format("2006/01/02 15:04:05"), done, name, hash, user)
				}

				rows.Close()

				for len(hashs) > 0 {
					i := -1
					fmt.Printf("select which log to mark: ")
					fmt.Scanf("%d", &i)

					if i < 0 {
						break
					}

					if i < len(hashs) {
						fmt.Printf("input mark> ")
						buf := make([]byte, 1024)
						if n, e := os.Stdout.Read(buf[:]); e == nil {

							arg("mark", strings.TrimSpace(string(buf[:n])))

							if _, e := db.Exec(fmt.Sprintf("update %s set mark=? where time = ? and name = ? and hash like '%s%%'", list, hashs[i]), arg("mark"), times[i], names[i]); e == nil {
							} else {
								fmt.Printf("%s\n", e)
							}
						}
					}
				}
			}
		}
	}
	return
}

// }}}

func fork() (e error) { // {{{
	fp := arg("srcfile")
	fn := arg("dstfile")

	var fr, fw *os.File
	if fr, e = os.Open(fp); e == nil {
		defer fr.Close()

		if fw, e = os.Create(fn); e == nil {
			defer fw.Close()

			io.Copy(fw, fr)

			arg("action", "fork")
			arg("srcfile", fn)
			arg("mark", fp)
			trace()

			arg("action", "fork")
			arg("srcfile", fp)
			arg("mark", fn)
			trace()
		}
	}
	return
}

// }}}
func move() (e error) { // {{{
	fp := arg("srcfile")
	fn := arg("dstfile")
	if e = os.Rename(fp, fn); e == nil {
		db.Exec(fmt.Sprintf("update name set name = ? where name = ?"), fn, fp)

		arg("srcfile", fn)
		if arg("action") == "" {
			arg("action", "move")
		}

		return trace()
	}
	return
}

// }}}

func trash() (e error) { // {{{
	arg("srcfile")
	arg("dstfile", path.Join(arg("trash"), fmt.Sprintf("%d-%s", time.Now().Unix(), path.Base(arg("srcfile")))))
	arg("action", "trash")
	return move()
}

// }}}
func clear() (e error) { // {{{
	var fl []os.FileInfo

	if fl, e = ioutil.ReadDir(arg("trash")); e == nil {
		names := make([]string, 0)

		for _, v := range fl {
			if v.IsDir() || v.Name()[0] == '.' {
				continue
			}
			names = append(names, v.Name())
			fmt.Printf("%d %s\n", len(names)-1, v.Name())
		}

		for len(names) > 0 {
			i := -1
			fmt.Printf("select which to clear: ")
			fmt.Scanf("%d", &i)

			if i < 0 {
				break
			}

			if i < len(names) && names[i] != "" {
				fp := path.Join(arg("trash"), names[i])
				arg("srcfile", fp)
				drop()

				os.Remove(fp)
				names[i] = ""

				log.Printf("[%s] clear %s", arg("mark"), arg("srcfile"))
			}
		}
	}
	return
}

// }}}
func restore() (e error) { // {{{
	var fl []os.FileInfo
	var rows *sql.Rows

	if fl, e = ioutil.ReadDir(arg("trash")); e == nil {
		var names = make([]string, 0)

		for _, v := range fl {
			if v.IsDir() || v.Name()[0] == '.' {
				continue
			}
			names = append(names, v.Name())
			fmt.Printf("%d %s\n", len(names)-1, v.Name())
		}

		for len(names) > 0 {
			i := -1
			fmt.Printf("select which file to restore: ")
			fmt.Scanf("%d", &i)

			if i < 0 {
				e = nil
				break
			}

			if i < len(names) && names[i] != "" {
				fp := path.Join(arg("trash"), names[i])
				fn := ""

				if rows, e = db.Query(fmt.Sprintf("select list from name where name = ?"), fp); e == nil && rows.Next() {
					var list string
					rows.Scan(&list)
					rows.Close()

					if rows, e = db.Query(fmt.Sprintf("select * from %s", list)); e == nil {
						var i, t int64
						var done, name, hash, user string
						var names = make([]string, 0)

						for i = 0; rows.Next(); i++ {
							rows.Scan(&t, &done, &name, &hash, &user)
							names = append(names, name)

							fmt.Printf("%d %s %s %s %s %s\n", len(names)-1, time.Unix(t, 0).Format("2006/01/02 15:04:05"), done, name, hash, user)
						}

						rows.Close()
						for len(names) > 0 {
							i := -1
							fmt.Printf("select which name to recover: ")
							fmt.Scanf("%d", &i)

							if i < 0 {
								break
							}

							fn = names[i]
							if _, e = os.Stat(fn); e != nil && names[i] != "" {
								arg("srcfile", fp)
								arg("dstfile", fn)
								arg("action", "restore")
								move()
								names[i] = ""

								log.Printf("[%s] restore %s", arg("mark"), arg("srcfile"))
								break
							}
						}
					}
				}
			}
		}
	}
	return
}

// }}}

var cmds = map[string]command{ // {{{
	"help":   command{"show share usage ", nil, []string{"cmd"}},
	"dump":   command{"dump markdown document of help", nil, []string{"dstfile"}},
	"listen": command{"start server to trace files' upload and download", listen, []string{"addr", "share", "log"}},

	"trace": command{"trace file", trace, []string{"srcfile", "mark"}},
	"drop":  command{"stop trace file", drop, []string{"srcfile"}},

	"show": command{"show file log", show, []string{"srcfile", "hash", "dstfile"}},
	"mark": command{"mark file log", mark, []string{"srcfile"}},

	"fork": command{"trace file's copying", fork, []string{"srcfile", "dstfile"}},
	"move": command{"trace file's moving", move, []string{"srcfile", "dstfile"}},

	"trash":   command{"trace file's trash", trash, []string{"srcfile"}},
	"clear":   command{"clear trash file", clear, nil},
	"restore": command{"restore trash file", restore, nil},
}

// }}}
var args = map[string]*argument{ // {{{
	"cmd": &argument{"sub command name", "help"},

	"action":  &argument{"trace file action", ""},
	"srcfile": &argument{"source file name", ""},
	"dstfile": &argument{"destination file name", ""},
	"hash":    &argument{"the hash value of file", ""},
	"mark":    &argument{"mark the file log", "127.0.0.1:9090"},

	"addr":  &argument{"socket listen address", ":9090"},
	"share": &argument{"share dirent path", fmt.Sprintf("%s/share", os.Getenv("HOME"))},
	"trash": &argument{"trash dirent path", fmt.Sprintf("%s/share/.trash", os.Getenv("HOME"))},

	"config": &argument{"config file name", fmt.Sprintf("%s/share/.trash/.share.conf", os.Getenv("HOME"))},
	"log":    &argument{"log file name", fmt.Sprintf("%s/share/.trash/.share.log", os.Getenv("HOME"))},

	"dbtype": &argument{"database software name", "sqlite3"},
	"dbfile": &argument{"trash database file", fmt.Sprintf("%s/share/.trash/.share.db", os.Getenv("HOME"))},
	"dbuser": &argument{"database user name", "share"},
	"dbword": &argument{"database pass word", "share"},
	"dbname": &argument{"database name", "share"},
}

// }}}
func arg(arg ...string) string { // {{{
	var a *argument

	if len(arg) > 0 {
		a = args[arg[0]]
	}

	if len(arg) > 1 {
		a.val = arg[1]
	}

	switch arg[0] {
	case "srcfile", "dstfile":
		if a.val != "" && !path.IsAbs(a.val) {
			pwd, _ := os.Getwd()
			a.val = path.Join(pwd, a.val)
		}
	case "share", "trash":
		if _, e := os.Stat(a.val); e != nil {
			os.MkdirAll(a.val, 0700)
		}
	}

	return a.val
}

// }}}

func dump() (e error) { // {{{
	var f *os.File
	if f, e = os.Create(arg("dstfile")); e != nil {
		return
	}

	f.Write([]byte(`## share
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
`))

	for k, v := range cmds {
		fmt.Fprintf(f, "### share %s", k)

		i := 0
		for _, vv := range v.args {
			i++
			fmt.Fprintf(f, " [%s", vv)
		}

		for i > 0 {
			fmt.Fprintf(f, "]")
			i--
		}

		fmt.Fprintf(f, " [args] \n%s\n", v.text)

		fmt.Fprintf(f, "\n")
		for _, vv := range v.args {
			i++
			a := args[vv]
			if a.val == "" || vv == "dstfile" {
				fmt.Fprintf(f, "* **%s** %s\n", vv, a.text)
			} else {
				fmt.Fprintf(f, "* **%s** (=%s) %s\n", vv, a.val, a.text)
			}
		}
		fmt.Fprintf(f, "\n")
	}

	fmt.Fprintf(f, "## appendix\n")
	fmt.Fprintf(f, "### other optional arguments\n")
	for k, v := range args {
		if v.val == "" || k == "dstfile" {
			fmt.Fprintf(f, "* **%s** %s\n", k, v.text)
		} else {
			fmt.Fprintf(f, "* **%s** (=%s) %s\n", k, v.val, v.text)
		}
	}

	fmt.Printf("dump markdown file '%s' success\n", arg("dstfile"))
	return
}

// }}}
func help() (e error) { // {{{
	if arg("cmd") == "help" {
		fmt.Printf("usage: share [subcommand] [arguments]\n")
		fmt.Printf("usage: share {help} {cmd} to show subcommand usage \n")

		fmt.Printf("\nusage: [subcommand] list\n")
		for k, v := range cmds {
			fmt.Printf("\t%s:\t%s\n", k, v.text)
		}

		fmt.Printf("\nusage: optional [arguments] list\n")
		for k, v := range args {
			if v.val == "" {
				fmt.Printf("\t%s: %s\n", k, v.text)
			} else {
				fmt.Printf("\t%s(=%s): %s\n", k, v.val, v.text)
			}
		}
	} else {
		if c, ok := cmds[arg("cmd")]; ok {
			fmt.Printf("usage: share {%s}", arg("cmd"))
			for _, v := range c.args {
				fmt.Printf(" [%s", v)
			}

			for i := 0; i < len(c.args); i++ {
				fmt.Printf("]")
			}
			fmt.Printf(" [key=val]\n")

			for _, v := range c.args {
				fmt.Printf("\t%s: %s\n", v, args[v].text)
			}

			fmt.Printf("\nusage: share {%s} other option arguments and default value\n", arg("cmd"))
			for k, v := range args {
				if v.val == "" {
					fmt.Printf("\t%s:\n\t\t%s\n", k, v.text)
				} else {
					fmt.Printf("\t%s(=%s):\n\t\t%s\n", k, v.val, v.text)
				}
			}
		} else {
			fmt.Printf("sub commnad %s not exists\n", arg("cmd"))
		}
	}
	return
}

// }}}
func main() { // {{{
	var sub string
	var cmd command
	var words []string

	buf := make([]byte, 1024)

	if len(os.Args) == 1 {
		fmt.Printf("usage: [subcommand] list\n")
		for k, v := range cmds {
			fmt.Printf("\t%s:\t%s\n", k, v.text)
		}
	}

	for {
		if len(os.Args) == 1 {
			fmt.Printf("share>")

			n, e := os.Stdin.Read(buf[:])
			if e != nil {
				break
			}

			words = strings.Fields(string(buf[0:n]))

			if n == 1 {
				continue
			}
		} else {
			words = os.Args[1:]
		}

		if len(words) > 0 {
			sub = words[0]
			cmd = cmds[sub]
			arg("cmd", sub)
		}

		if len(words) > 1 {
			for i, v := range words[1:] {

				if pos := strings.Index(v, "="); pos > -1 {
					arg(v[0:pos], v[pos+1:])
				} else {
					if cmd.args != nil && i < len(cmd.args) {
						arg(cmd.args[i], v)
					}
				}
			}
		}

		if _, e := os.Stat(arg("config")); e == nil {
			if f, e := os.Open(arg("config")); e == nil {
				rd := bufio.NewReader(f)
				for {
					if l, e := rd.ReadString('\n'); e == nil {
						l = strings.TrimSpace(l)

						if pos := strings.Index(l, "="); pos > -1 {
							arg(l[0:pos], l[pos+1:])
							fmt.Printf("%s=%s\n", l[0:pos], arg(l[0:pos]))
						}
					} else {
						break
					}
				}
			}
		}

		if cmd.hand != nil {

			switch arg("dbtype") {
			case "sqlite3":
				db, _ = sql.Open("sqlite3", arg("dbfile"))
			case "mysql":
				db, _ = sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", arg("dbuser"), arg("dbword"), arg("dbname")))
			default:
				fmt.Printf("dbtype error, choice mysql3 or mysql\n")
				break
			}

			db.Exec("create table if not exists hash(time int, hash char(32) primary key, size int, count int)")
			db.Exec("create table if not exists name(time int, name char(255) primary key, list char(8))")
			db.Exec("create table if not exists config(name char(32) primary key, value text)")
			db.Exec("insert into config values('count', 0)")

			if f, e := os.OpenFile(arg("log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); e == nil {
				log.SetOutput(f)
			}

			if e := cmd.hand(); e != nil {
				fmt.Printf("%s", e)
			}
		} else {
			switch sub {
			case "dump":
				dump()
			default:
				help()
			}
		}

		if len(os.Args) != 1 {
			break
		}
	}
}

// }}}
