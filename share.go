package main // {{{
// }}}
import ( // {{{
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
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

	if r.Method == "GET" {
		fs, e := os.Stat("." + r.URL.Path)
		if e != nil {
			w.Write([]byte("erorr"))
			return
		}

		if fs.IsDir() {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(fmt.Sprintf("<DOCTYPE html><head><meta name='viewport' content='width=device-width, initial-scale=1.0'></head><body><form method='POST' action='%s' enctype='multipart/form-data'><input type='file' name='file'><input type='submit'></form></body>", r.URL.Path)))

			w.Write([]byte(fmt.Sprintf("<a href='/'>home: /</a> ")))
			back := r.URL.Path[0 : len(r.URL.Path)-1]
			back = back[0 : strings.LastIndex(back, "/")+1]
			w.Write([]byte(fmt.Sprintf("<a href='%s'>back: %s</a> ", back, back)))
			w.Write([]byte(fmt.Sprintf("path: %s<hr>", r.URL.Path)))

			log.Printf("[%s] %s %s\n", r.RemoteAddr, r.Method, r.URL)

			if fl, e := ioutil.ReadDir(fs.Name()); e == nil {
				w.Write([]byte(fmt.Sprintf("<pre>")))
				for i, v := range fl {
					if v.IsDir() {
						w.Write([]byte(fmt.Sprintf("%d %s    ---   <a href='%s/'> %s</a><br>", i, v.ModTime().Format("2006-01-02 15:04:05"), v.Name(), v.Name())))
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

						w.Write([]byte(fmt.Sprintf("%d %s %6s   <a href='%s'> %s</a><br>", i, v.ModTime().Format("2006-01-02 15:04:05"), size, v.Name(), v.Name())))
					}
				}
				w.Write([]byte(fmt.Sprintf("</pre>")))
			}
		} else {
			var name string = "." + r.URL.Path
			var pwd, _ = os.Getwd()
			name = path.Join(pwd, name)

			if f, e := os.Open(name); e == nil {
				defer f.Close()

				io.Copy(w, f)

				arg("srcfile", name)
				arg("action", "GET")
				arg("remote", r.RemoteAddr)
				trace()
			}
		}
	}

	if r.Method == "POST" {
		if u, h, e := r.FormFile("file"); e == nil {
			defer u.Close()

			var name string = "." + r.URL.Path + h.Filename
			var pwd, _ = os.Getwd()
			name = path.Join(pwd, name)

			if info, e := os.Stat(name); e == nil {
				log.Printf("%s already exists\n", info.Name())
				w.Write([]byte(fmt.Sprintf("%s already exists\n", info.Name())))
				return
			}

			if f, e := os.Create(name); e == nil {
				defer f.Close()

				u.Seek(0, os.SEEK_SET)
				io.Copy(f, u)
				w.Write([]byte(fmt.Sprintf("%s upload success\n", name)))

				arg("srcfile", name)
				arg("action", "POST")
				arg("remote", r.RemoteAddr)
				trace()
			}
		}
	}
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
	f, e := os.Open(file)
	if e != nil {
		return "", 0
	}

	h := md5.New()
	io.Copy(h, f)
	md := hex.EncodeToString(h.Sum(nil))

	os.MkdirAll(path.Join(arg("trash"), md[0:2]), 0700)

	fm, e := os.Create(path.Join(arg("trash"), md[0:2], md))
	f.Seek(0, os.SEEK_SET)
	size, _ := io.Copy(fm, f)

	return md, size
}

// }}}
func trace() int { // {{{
	fp := arg("srcfile")
	action := arg("action")
	md, size := filemd(fp)

	if !path.IsAbs(fp) {
		pwd, _ := os.Getwd()
		fp = path.Join(pwd, fp)
	}

	if rows, e := db.Query("select list from name where name=?", fp); e == nil {
		defer rows.Close()
		if rows.Next() {
			var list string
			rows.Scan(&list)

			if rows, e := db.Query(fmt.Sprintf("select done, name, hash from %s order by time desc limit 1", list)); e == nil && rows.Next() {
				defer rows.Close()
				var done, name, hash string
				rows.Scan(&done, &name, &hash)

				if action != done || name != fp || md != hash {
					db.Exec(fmt.Sprintf("insert into %s values(?, ?, ?, ?, ?)", list), time.Now().Unix(), action, fp, md, arg("remote"))

					r, e := db.Exec(fmt.Sprintf("update hash set count=count+1 where hash = ?"), md)
					n, e := r.RowsAffected()

					if n == 0 {
						db.Exec(fmt.Sprintf("insert into hash values(%d, '%s', %d, 1)", time.Now().Unix(), md, size))
					}

					fmt.Printf("%s\n", e)
				}
			}
		} else {
			count := 0
			if rows, _ := db.Query("select value from config where name='count'"); rows.Next() {
				defer rows.Close()
				rows.Scan(&count)
				count++
				db.Exec("update config set value=? where name='count'", count)
			}

			db.Exec(fmt.Sprintf("insert into name values(%d, '%s', 'file%d')", time.Now().Unix(), fp, count))
			db.Exec(fmt.Sprintf("create table if not exists file%d(time int, done char(8), name text, hash char(32), mark text)", count))
			db.Exec(fmt.Sprintf("insert into file%d values(?, ?, ?, ?, ?)", count), time.Now().Unix(), action, fp, md, arg("remote"))

			r, e := db.Exec(fmt.Sprintf("update hash set count=count+1 where hash = ?"), md)
			n, e := r.RowsAffected()

			if n == 0 {
				db.Exec(fmt.Sprintf("insert into hash values(%d, '%s', %d, 1)", time.Now().Unix(), md, size))
			}

			fmt.Printf("%d %s\n", n, e)
		}
	}
	log.Printf("[%s] %s %s %s", arg("remote"), action, fp, md)
	return 1
}

// }}}
func fork() int { // {{{
	fp := arg("srcfile")
	if !path.IsAbs(fp) {
		pwd, _ := os.Getwd()
		fp = path.Join(pwd, fp)
	}

	fn := arg("dstfile")
	if !path.IsAbs(fn) {
		pwd, _ := os.Getwd()
		fn = path.Join(pwd, fn)
	}

	fr, _ := os.Open(fp)
	fw, _ := os.Create(fn)
	io.Copy(fw, fr)

	arg("action", "fork")
	arg("srcfile", fn)
	trace()
	return 1
}

// }}}
func move() int { // {{{
	fp := arg("srcfile")
	if !path.IsAbs(fp) {
		pwd, _ := os.Getwd()
		fp = path.Join(pwd, fp)
	}

	fn := arg("dstfile")
	if !path.IsAbs(fn) {
		pwd, _ := os.Getwd()
		fn = path.Join(pwd, fn)
	}

	os.Rename(fp, fn)
	db.Exec(fmt.Sprintf("update name set name = ? where name = ?"), fn, fp)

	arg("action", "move")
	arg("srcfile", fn)
	trace()
	return 1
}

// }}}
func trash() int { // {{{
	fp := arg("srcfile")
	if !path.IsAbs(fp) {
		pwd, _ := os.Getwd()
		fp = path.Join(pwd, fp)
	}

	fn := fmt.Sprintf("%s/trash/%s", os.Getenv("HOME"), path.Base(fp))
	os.Rename(fp, fn)
	db.Exec(fmt.Sprintf("update name set name = ? where name = ?"), fn, fp)

	arg("action", "trash")
	arg("srcfile", fn)
	trace()
	return 1
}

// }}}

func drop() int { // {{{
	if rows, e := db.Query(fmt.Sprintf("select list from name where name = ?"), arg("srcfile")); e == nil && rows.Next() {
		defer rows.Close()

		var list string
		rows.Scan(&list)

		if rows, e := db.Query(fmt.Sprintf("select hash from %s", list)); e == nil {
			defer rows.Close()

			var hash string
			for rows.Next() {
				rows.Scan(&hash)
				db.Exec(fmt.Sprintf("update hash set count=count-1 where hash=?"), hash)
				db.Exec(fmt.Sprintf("delete from name where name = ?"), arg("srcfile"))
				db.Exec(fmt.Sprintf("drop table %s", list))
			}
		}
	}

	if rows, e := db.Query(fmt.Sprintf("select hash from hash where count=0")); e == nil {
		defer rows.Close()

		var hash string
		for rows.Next() {
			rows.Scan(&hash)
			os.Remove(path.Join(arg("trash"), hash[0:2], hash))
		}

		db.Exec(fmt.Sprintf("delete from hash where count = 0"))
	}

	return 1
}

// }}}
func show() int { // {{{

	if arg("srcfile") == "" {
		if rows, e := db.Query(fmt.Sprintf("select * from name")); e == nil {
			defer rows.Close()

			var t int64
			var name, list string

			for rows.Next() {
				rows.Scan(&t, &name, &list)
				fmt.Printf("%s %s %s\n", time.Unix(t, 0).Format("2006/01/02 15:04:05"), name, list)
			}
		}
		return 1
	}

	if rows, e := db.Query(fmt.Sprintf("select list from name where name = ?"), arg("srcfile")); e == nil && rows.Next() {
		defer rows.Close()

		var list string
		rows.Scan(&list)

		if rows, e := db.Query(fmt.Sprintf("select * from %s where hash like '%s%%'", list, arg("hash"))); e == nil {
			defer rows.Close()

			var i, t int64
			var done, name, hash, user string

			for i = 0; rows.Next(); i++ {
				rows.Scan(&t, &done, &name, &hash, &user)

				fmt.Printf("%s %s %s %s %s\n", time.Unix(t, 0).Format("2006/01/02 15:04:05"), done, name, hash, user)
			}

			if arg("dstfile") != "" {
				f, _ := os.Open(path.Join(arg("trash"), hash[0:2], hash))
				df, _ := os.Create(arg("dstfile"))
				io.Copy(df, f)
				df.Close()
				f.Close()
			}
		}
	}
	return 1
}

// }}}
func mark() int { // {{{
	if rows, e := db.Query(fmt.Sprintf("select list from name where name = ?"), arg("srcfile")); e == nil && rows.Next() {
		defer rows.Close()

		var list string
		rows.Scan(&list)

		println(list)
		if _, e := db.Exec(fmt.Sprintf("update %s set mark=? where hash like '%s%%'", list, arg("hash")), arg("mark")); e == nil {
			println(fmt.Sprintf("update %s set mark=? where hash like '%s%%'", list, arg("hash")), arg("mark"))
			println(list)
		}
	}

	return 1
}

// }}}

func dump() int {
	if arg("dstfile") != "" {
		f, _ := os.Create(arg("dstfile"))
		f.Write([]byte(`## share
to automate personal files's protection, something like git 用类似于git的方式，自动化保护个人文件

usage: share [subcommand] [arguments]

arguments has two style: indexed and named

indexed arguments have only value, different position has different meanings

named arguments have both name and value, like name=value

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

		fmt.Fprintf(f, "### other optional arguments\n\n")
		for k, v := range args {
			if v.val == "" || k == "dstfile" {
				fmt.Fprintf(f, "* **%s** %s\n", k, v.text)
			} else {
				fmt.Fprintf(f, "* **%s** (=%s) %s\n", k, v.val, v.text)
			}
		}
	}
	return 1
}

var cmds = map[string]command{ // {{{
	"dump":   command{"dump help document", nil, []string{"dstfile"}},
	"help":   command{"show share usage help", nil, []string{"cmd"}},
	"listen": command{"socket listen address", listen, []string{"addr", "share", "log"}},

	"trace": command{"begin to trace file", trace, []string{"srcfile"}},
	"fork":  command{"copy file ", fork, []string{"srcfile", "dstfile"}},
	"move":  command{"move file ", move, []string{"srcfile", "dstfile"}},
	"trash": command{"move file to trash", trash, []string{"srcfile"}},

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
	"remote":  &argument{"socket remote addr", "127.0.0.1:9090"},
	"hash":    &argument{"the hash value of file", ""},
	"mark":    &argument{"mark the file log", ""},

	"addr":  &argument{"socket listen address", ":9090"},
	"share": &argument{"share dirent path", "./temp"},
	"trash": &argument{"trash dirent path", "./trash"},
	"log":   &argument{"trash dirent path", "./share.log"},

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
	}

	return a.val
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
	return 1
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
		db, _ = sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", arg("dbuser"), arg("dbword"), arg("dbname")))
		db.Exec("create table if not exists hash(time int, hash char(32) primary key, size int, count int)")
		db.Exec("create table if not exists name(time int, name char(255) primary key, list char(8))")
		db.Exec("create table if not exists config(name char(32) primary key, value text)")
		db.Exec("insert into config values('count', 0)")

		os.Exit(cmd.hand())
	} else {
		switch sub {
		case "help":
			os.Exit(help())
		case "dump":
			os.Exit(dump())
		}
	}
}

// }}}
