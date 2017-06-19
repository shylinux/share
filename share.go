package main // {{{
// }}}
import ( // {{{
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	// _ "github.com/go-sql-driver/mysql"
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
	hand func() int
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
			fmt.Fprintf(w, `<body><form onsubmit='if (this.mark.value == "") {alert("must add comment"); return false} else {return true}' method='POST' action='%s' enctype='multipart/form-data'><input type='file' name='file'><input type='text' name='mark'><input type='submit'></form>`, r.URL.Path)

			fmt.Fprintf(w, "<pre><a href='/'>home: /</a>   ")
			back := r.URL.Path[0 : len(r.URL.Path)-1]
			back = back[0 : strings.LastIndex(back, "/")+1]
			fmt.Fprintf(w, "<a href='%s'>back: %s</a>   ", back, back)
			fmt.Fprintf(w, "path: %s<hr></pre>", r.URL.Path)

			if fl, e := ioutil.ReadDir(fs.Name()); e == nil {
				fmt.Fprintf(w, "<pre>")
				for i, v := range fl {
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
			}
		} else {
			if f, e := os.Open(arg("srcfile", "."+r.URL.Path)); e == nil {
				defer f.Close()

				io.Copy(w, f)

				arg("action", "GET")
				arg("mark", r.RemoteAddr)
				trace()
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
func listen() int { // {{{

	if l, e := os.OpenFile(arg("log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600); e == nil {
		log.SetOutput(l)
	}

	for k, v := range args {
		log.Printf("\t%s=%s\t%s\n", k, v.val, v.text)
	}

	os.Chdir(arg("share"))

	http.HandleFunc("/", index)

	http.ListenAndServe(arg("addr"), nil)
	return 1
}

// }}}

func filemd(file string) (string, int64) { // {{{
	var e error

	if f, e := os.Open(file); e == nil {
		defer f.Close()

		h := md5.New()
		io.Copy(h, f)
		md := hex.EncodeToString(h.Sum(nil))

		if e = os.MkdirAll(path.Join(arg("trash"), md[0:2]), 0700); e == nil {
			if fm, e := os.Create(path.Join(arg("trash"), md[0:2], md)); e == nil {
				defer fm.Close()

				f.Seek(0, os.SEEK_SET)
				size, _ := io.Copy(fm, f)

				return md, size
			}
		}
	}

	log.Printf("%s\n", e)
	return "", 0
}

// }}}
func trace() int { // {{{
	fp := arg("srcfile")
	action := arg("action")
	md, size := filemd(fp)

	if fp == "" || action == "" || md == "" || size == 0 {
		log.Printf("filename: %s\n", fp)
		log.Printf("action: %s\n", action)
		log.Printf("md: %s\n", md)
		log.Printf("size: %d\n", size)
		log.Printf("error something is null\n")
		return 1
	}

	sql := "select list from name where name=?"
	if rows, e := db.Query(sql, fp); e == nil {
		if rows.Next() {
			var list string
			rows.Scan(&list)
			rows.Close()

			sql := fmt.Sprintf("select done, name, hash from %s order by time desc limit 1", list)
			if rows, e := db.Query(sql); e == nil && rows.Next() {
				var done, name, hash string
				rows.Scan(&done, &name, &hash)
				rows.Close()

				if action != done || name != fp || md != hash {
					db.Exec(fmt.Sprintf("insert into %s values(?, ?, ?, ?, ?)", list), time.Now().Unix(), action, fp, md, arg("mark"))

					db.Exec(fmt.Sprintf("insert into hash values(%d, '%s', %d, 0)", time.Now().Unix(), md, size))
					db.Exec(fmt.Sprintf("update hash set count=count+1 where hash = ?"), md)
				}
			} else {
				log.Printf("sql exec error: %s\n", sql)
				return 1
			}
		} else {
			rows.Close()

			count := 0
			sql := "select value from config where name='count'"
			if rows, e := db.Query(sql); e == nil && rows.Next() {
				rows.Scan(&count)
				rows.Close()

				db.Exec("update config set value=value+1 where name='count'")
			} else {
				log.Printf("sql exec error:%s \n", sql)
				return 1
			}

			db.Exec(fmt.Sprintf("insert into name values(%d, '%s', 'file%d')", time.Now().Unix(), fp, count))

			db.Exec(fmt.Sprintf("create table if not exists file%d(time int, done char(8), name text, hash char(32), mark text)", count))
			db.Exec(fmt.Sprintf("insert into file%d values(?, ?, ?, ?, ?)", count), time.Now().Unix(), action, fp, md, arg("mark"))

			db.Exec(fmt.Sprintf("insert into hash values(%d, '%s', %d, 0)", time.Now().Unix(), md, size))
			db.Exec(fmt.Sprintf("update hash set count=count+1 where hash = ?"), md)
		}
	} else {
		log.Printf("%s\n", e)
		return 1
	}

	log.Printf("[%s] %s %s %s", arg("mark"), action, fp, md)
	return 0
}

// }}}
func fork() int { // {{{
	var e error

	fp := arg("srcfile")
	fn := arg("dstfile")
	if fp == "" || fn == "" {
		fmt.Printf("srcfile: %s\n", fp)
		fmt.Printf("dstfile: %s\n", fn)
		fmt.Printf("error some thing is null\n")
		return 1
	}

	if fr, e := os.Open(fp); e == nil {
		defer fr.Close()

		if fw, e := os.Create(fn); e == nil {
			defer fw.Close()

			io.Copy(fw, fr)

			arg("action", "fork")
			arg("srcfile", fn)
			return trace()
		}

	}

	log.Printf("%s\n", e)
	return 1
}

// }}}
func move() int { // {{{
	var e error
	fp := arg("srcfile")
	fn := arg("dstfile")
	if fp == "" || fn == "" {
		fmt.Printf("srcfile: %s\n", fp)
		fmt.Printf("dstfile: %s\n", fn)
		fmt.Printf("error some thing is null\n")
		return 1
	}

	if e = os.Rename(fp, fn); e == nil {
		db.Exec(fmt.Sprintf("update name set name = ? where name = ?"), fn, fp)

		arg("action", "move")
		arg("srcfile", fn)
		return trace()

	}

	log.Printf("%s\n", e)
	return 1
}

// }}}
func trash() int { // {{{
	var e error
	fp := arg("srcfile")
	fn := path.Join(arg("trash"), fmt.Sprintf("%d-%s", time.Now().Unix(), path.Base(fp)))
	if _, e := os.Stat(fn); e == nil || fp == "" {
		fmt.Printf("srcfile: %s\n", fp)
		fmt.Printf("dstfile: %s\n", fn)
		fmt.Printf("error some thing is null\n")
		return 1
	}

	if e = os.Rename(fp, fn); e == nil {
		db.Exec(fmt.Sprintf("update name set name = ? where name = ?"), fn, fp)

		arg("action", "trash")
		arg("srcfile", fn)
		return trace()
	}

	log.Printf("%s\n", e)
	return 1
}

// }}}

func drop() int { // {{{
	var e error
	fp := arg("srcfile")
	if fp == "" {
		fmt.Printf("srcfile: %s\n", fp)
		fmt.Printf("error some thing is null\n")
		return 1
	}

	if rows, e := db.Query(fmt.Sprintf("select list from name where name = ?"), arg("srcfile")); e == nil && rows.Next() {
		var list string
		rows.Scan(&list)
		rows.Close()

		if rows, e := db.Query(fmt.Sprintf("select hash from %s", list)); e == nil {

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

			db.Exec(fmt.Sprintf("delete from name where name = ?"), arg("srcfile"))
			db.Exec(fmt.Sprintf("drop table %s", list))

			if rows, e := db.Query(fmt.Sprintf("select hash from hash where count=0")); e == nil {

				var hash string
				for rows.Next() {
					rows.Scan(&hash)
					os.Remove(path.Join(arg("trash"), hash[0:2], hash))
				}
				rows.Close()

				db.Exec(fmt.Sprintf("delete from hash where count = 0"))
			}

			return 0
		}
	}

	log.Printf("%s\n", e)
	return 1
}

// }}}
func show() int { // {{{
	var e error
	if arg("srcfile") == "" {
		if rows, e := db.Query(fmt.Sprintf("select * from name")); e == nil {
			var t int64
			var name, list string

			for rows.Next() {
				rows.Scan(&t, &name, &list)
				fmt.Printf("%s %s %s\n", time.Unix(t, 0).Format("2006/01/02 15:04:05"), name, list)
			}

			rows.Close()
			return 0
		} else {
			log.Printf("%s\n", e)
			return 1
		}
	}

	if rows, e := db.Query(fmt.Sprintf("select list from name where name = ?"), arg("srcfile")); e == nil && rows.Next() {
		var list string
		rows.Scan(&list)
		rows.Close()

		if rows, e := db.Query(fmt.Sprintf("select * from %s where hash like '%s%%'", list, arg("hash"))); e == nil {
			var i, t int64
			var done, name, hash, user string

			for i = 0; rows.Next(); i++ {
				rows.Scan(&t, &done, &name, &hash, &user)

				fmt.Printf("%s %s %s %s %s\n", time.Unix(t, 0).Format("2006/01/02 15:04:05"), done, name, hash, user)
			}

			rows.Close()

			if arg("dstfile") == "" {
				return 0
			}

			if f, e := os.Open(path.Join(arg("trash"), hash[0:2], hash)); e == nil {
				defer f.Close()

				if df, e := os.Create(arg("dstfile")); e == nil {
					defer df.Close()

					io.Copy(df, f)
					return 0
				}
			}
		}
	}

	log.Printf("%s\n", e)
	return 1
}

// }}}
func mark() int { // {{{
	var e error
	fp := arg("srcfile")
	md := arg("hash")
	m := arg("mark")
	if fp == "" || md == "" || m == "" {
		fmt.Printf("srcfile: %s\n", fp)
		fmt.Printf("md5: %s\n", md)
		fmt.Printf("mark: %s\n", m)
		fmt.Printf("error some thing is null\n")
		return 1
	}

	if rows, e := db.Query(fmt.Sprintf("select list from name where name = ?"), arg("srcfile")); e == nil && rows.Next() {
		var list string
		rows.Scan(&list)
		rows.Close()

		if _, e := db.Exec(fmt.Sprintf("update %s set mark=? where name = %s and hash like '%s%%'", list, fp, md), arg("mark")); e == nil {
			return 0
		}
	}

	log.Printf("%s\n", e)
	return 1
}

// }}}

var cmds = map[string]command{ // {{{
	"help":   command{"show share usage ", nil, []string{"cmd"}},
	"dump":   command{"dump markdown document of help", nil, []string{"dstfile"}},
	"listen": command{"start server to trace files' transport", listen, []string{"addr", "share", "log"}},

	"trace": command{"trace file", trace, []string{"srcfile", "mark"}},
	"fork":  command{"trace file's copying", fork, []string{"srcfile", "dstfile", "mark"}},
	"move":  command{"trace file's moving", move, []string{"srcfile", "dstfile"}},
	"trash": command{"trace file's trash", trash, []string{"srcfile"}},

	"drop": command{"stop trace file", drop, []string{"srcfile"}},
	"show": command{"show file log", show, []string{"srcfile", "hash", "dstfile"}},
	"mark": command{"mark file log", mark, []string{"srcfile", "hash", "mark"}},
}

// }}}
var args = map[string]*argument{ // {{{
	"cmd": &argument{"sub command name", "help"},

	"action":  &argument{"trace file action", "trace"},
	"srcfile": &argument{"source file name", ""},
	"dstfile": &argument{"destination file name", ""},
	"hash":    &argument{"the hash value of file", ""},
	"mark":    &argument{"mark the file log", "127.0.0.1:9090"},

	"addr":   &argument{"socket listen address", ":9090"},
	"share":  &argument{"share dirent path", fmt.Sprintf("%s/share", os.Getenv("HOME"))},
	"trash":  &argument{"trash dirent path", fmt.Sprintf("%s/share/.trash", os.Getenv("HOME"))},
	"log":    &argument{"trash log file", fmt.Sprintf("%s/share/.trash/.share.log", os.Getenv("HOME"))},
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
func dump() int { // {{{
	if arg("dstfile") != "" {
		f, e := os.Create(arg("dstfile"))
		if e != nil {
			fmt.Printf("%s\n", e)
			return 1
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
	} else {
		fmt.Printf("dump markdown file name invalidate\n")
		return 1
	}
	return 0
}

// }}}
func help() int { // {{{
	if arg("cmd") == "help" {
		fmt.Printf("usage: share [subcommand] [argument]\n")
		fmt.Printf("\t[command] sub command\n")
		fmt.Printf("\t[argument] sub command argument\n")
		fmt.Printf("\nusage: sub [command] list\n")
		for k, v := range cmds {
			fmt.Printf("\t%s:\t%s\n", k, v.text)
		}
	} else {
		if c, ok := cmds[arg("cmd")]; ok {
			fmt.Printf("sub commnad [%s] indexed args list \n", arg("cmd"))
			for i, v := range c.args {
				fmt.Printf("\t%d:%s\n", i, v)
			}

			fmt.Printf("\nsub commnad [%s] named args list \n", arg("cmd"))
			for k, v := range args {
				fmt.Printf("\t%s=%s\t%s\n", k, v.val, v.text)
			}
		} else {
			fmt.Printf("sub commnad %s not exists\n", arg("cmd"))
		}
	}
	return 0
}

// }}}
func main() { // {{{
	var sub string
	var cmd command

	if len(os.Args) > 1 {
		sub = os.Args[1]
		cmd = cmds[sub]
		arg("cmd", sub)
	}

	if len(os.Args) > 2 {
		for i, v := range os.Args {
			if i < 2 {
				continue
			}

			if pos := strings.Index(os.Args[i], "="); pos > -1 {
				arg(v[0:pos], v[pos+1:])
			} else {
				if cmd.args != nil && i-2 < len(cmd.args) {
					arg(cmd.args[i-2], v)
				}
			}
		}
	}

	if cmd.hand != nil {
		// db, _ = sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", arg("dbuser"), arg("dbword"), arg("dbname")))
		db, _ = sql.Open("sqlite3", arg("dbfile"))
		db.Exec("create table if not exists hash(time int, hash char(32) primary key, size int, count int)")
		db.Exec("create table if not exists name(time int, name char(255) primary key, list char(8))")
		db.Exec("create table if not exists config(name char(32) primary key, value text)")
		db.Exec("insert into config values('count', 0)")

		if f, e := os.OpenFile(arg("log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); e == nil {
			log.SetOutput(f)
		}

		os.Exit(cmd.hand())
	} else {
		switch sub {
		case "dump":
			os.Exit(dump())
		default:
			os.Exit(help())
		}
	}
}

// }}}
