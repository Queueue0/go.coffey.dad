package models

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type Post struct {
	ID          int
	Title       string
	Body        template.HTML
	Tags        TagList
	Created     time.Time
	Modified    time.Time
	IsDraft     bool
	URL         string
	Description string
	HeaderImage Image
}

type Image struct {
	Location string
	Mime     string
}

type PostModel struct {
	DB *sql.DB
}

func (m *PostModel) parsePostRow(s Scanner, parse bool) (Post, error) {
	var p Post
	var imgPath string

	err := s.Scan(&p.ID, &p.Title, &p.Body, &p.Created, &p.Modified, &p.IsDraft, &p.URL, &imgPath, &p.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Post{}, ErrNoRecord
		} else {
			return Post{}, err
		}
	}

	tags, err := m.AllTagsForPost(p.ID)
	if err != nil {
		return Post{}, err
	}

	p.Tags = tags

	var mime string
	img, err := os.ReadFile("." + imgPath)
	// We have a default to fall back on, so we try to do that instead of
	// returning an error
	if err != nil {
		imgPath = "/static/img/bad_placeholder.png"
		mime = "image/png"
	} else {
		mime = http.DetectContentType(img)
	}

	p.HeaderImage = Image{
		Location: imgPath,
		Mime:     mime,
	}

	if parse {
		p.ParseBody()
	}

	return p, nil
}

func (p *Post) ParseBody() {
	body := []byte(p.Body)
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	pr := parser.NewWithExtensions(extensions)
	doc := pr.Parse(body)

	htmlFlags := html.CommonFlags
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	p.Body = template.HTML(markdown.Render(doc, renderer))
}

func (m *PostModel) Insert(p Post) (int, error) {
	stmt := "INSERT INTO post (title, body, is_draft, created, modified, url, header_image_path, meta_description) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"

	result, err := m.DB.Exec(stmt, p.Title, p.Body, p.IsDraft, p.Created, p.Modified, p.URL, p.HeaderImage.Location, p.Description)
	if err != nil {
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "post_uc_url") {
				return 0, ErrDuplicateUrl
			}
		}
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	p.ID = int(id)
	err = m.UpdateTags(p)
	if err != nil {
		return p.ID, err
	}

	return p.ID, nil
}

func (m *PostModel) Update(p Post) error {
	stmt := "UPDATE post SET title=?, body=?, is_draft=?, url=?, header_image_path=?, meta_description=?, created=?, modified=? WHERE id=?"

	result, err := m.DB.Exec(stmt, p.Title, p.Body, p.IsDraft, p.URL, p.HeaderImage.Location, p.Description, p.Created, p.Modified, p.ID)
	if err != nil {
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "post_uc_url") {
				return ErrDuplicateUrl
			}
		}
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	err = m.UpdateTags(p)
	if err != nil {
		return err
	}

	return nil
}

func (m *PostModel) UpdateTags(p Post) error {
	oldTags, err := m.AllTagsForPost(p.ID)
	if err != nil {
		return err
	}

	// (t)o (b)e (r)emoved
	var tbr TagList
	for _, t := range oldTags {
		if !p.Tags.Contains(t) {
			tbr = append(tbr, t)
		}
	}

	for _, t := range tbr {
		r, err := m.DeletePostTag(p.ID, t.ID)
		if err != nil {
			return err
		}

		if r != 1 {
			return errors.New(fmt.Sprintf("Unexpected number of deleted rows: %d", r))
		}
	}

	for _, t := range p.Tags {
		tid, err := m.InsertTagIfNotExists(t)
		if err != nil {
			return err
		}

		t.ID = tid

		err = m.InsertPostTagIfNotExists(p, t)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *PostModel) TogglePublishStatus(id int) error {
	stmt := "UPDATE post SET is_draft = NOT is_draft WHERE id = ?"

	result, err := m.DB.Exec(stmt, id)
	if err != nil {
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (m *PostModel) Get(id int) (Post, error) {
	stmt := `SELECT id, title, body, created, modified, is_draft, url, header_image_path, meta_description FROM post
	WHERE id=?`

	row := m.DB.QueryRow(stmt, id)
	return m.parsePostRow(row, false)
}

func (m *PostModel) GetByURL(url string) (Post, error) {
	stmt := `SELECT id, title, body, created, modified, is_draft, url, header_image_path, meta_description FROM post
	WHERE url=?`

	row := m.DB.QueryRow(stmt, url)
	return m.parsePostRow(row, false)
}

func (m *PostModel) Latest(n int) ([]Post, error) {
	stmt := `SELECT id, title, body, created, modified, is_draft, url, header_image_path, meta_description FROM post
	WHERE is_draft = FALSE ORDER BY created DESC LIMIT ?`

	rows, err := m.DB.Query(stmt, n)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []Post

	for rows.Next() {
		p, err := m.parsePostRow(rows, true)
		if err != nil {
			return nil, err
		}

		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (m *PostModel) Filter(isDraft bool, tagName string) ([]Post, error) {
	posts, err := m.All()
	if err != nil {
		return nil, err
	}

	t := Tag{Name: tagName}
	fPosts := []Post{}
	for _, p := range posts {
		if p.Tags.Contains(t) {
			fPosts = append(fPosts, p)
		}
	}

	return fPosts, nil
}

func (m *PostModel) All() ([]Post, error) {
	stmt := `SELECT id, title, body, created, modified, is_draft, url, header_image_path, meta_description FROM post
	WHERE is_draft = FALSE ORDER BY created DESC`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []Post

	for rows.Next() {
		p, err := m.parsePostRow(rows, true)
		if err != nil {
			return nil, err
		}

		posts = append(posts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (m *PostModel) PublishDraft(id int) (int, error) {
	stmt := "UPDATE post SET is_draft = TRUE WHERE id = ?"

	result, err := m.DB.Exec(stmt, id)
	if err != nil {
		return 0, err
	}

	postId, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(postId), nil
}

func (m *PostModel) AllDrafts() ([]Post, error) {
	stmt := `SELECT id, title, body, created, modified, is_draft, url, header_image_path, meta_description FROM post
       WHERE is_draft = TRUE ORDER BY id DESC`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var drafts []Post

	for rows.Next() {
		p, err := m.parsePostRow(rows, true)
		if err != nil {
			return nil, err
		}

		drafts = append(drafts, p)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return drafts, nil
}
