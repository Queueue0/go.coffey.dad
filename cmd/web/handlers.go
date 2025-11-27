package main

import (
	"errors"
	"fmt"
	"html/template"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"coffey.dad/internal/models"
	"coffey.dad/internal/validator"

	"github.com/julienschmidt/httprouter"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	posts, err := app.posts.Latest(5)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := app.newTemplateData(r)
	data.Posts = posts

	app.render(w, r, http.StatusOK, "home.tmpl", data)
}

func (app *application) postView(w http.ResponseWriter, r *http.Request) {
	var post models.Post
	params := httprouter.ParamsFromContext(r.Context())
	idparam := params.ByName("id")

	id, err := strconv.Atoi(idparam)
	if err != nil {
		post, err = app.posts.GetByURL(idparam)
		if err != nil {
			if errors.Is(err, models.ErrNoRecord) {
				app.notFound(w, r)
			} else {
				app.serverError(w, r, err)
			}
			return
		}
	} else {
		post, err = app.posts.Get(id)
		if err != nil {
			if errors.Is(err, models.ErrNoRecord) {
				app.notFound(w, r)
			} else {
				app.serverError(w, r, err)
			}
			return
		}
	}

	if post.IsDraft {
		app.notFound(w, r)
		return
	}

	post.ParseBody()

	flash := app.sessionManager.PopString(r.Context(), "flash")
	data := app.newTemplateData(r)
	data.Post = post
	data.Flash = flash

	app.logger.Info("Post Info", "Header Path", post.HeaderImage.Location, "Header MIME", post.HeaderImage.Mime)

	app.render(w, r, http.StatusOK, "post_view.tmpl", data)
}

func (app *application) postList(w http.ResponseWriter, r *http.Request) {
	filter, err := url.PathUnescape(r.URL.Query().Get("filter"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var posts []models.Post
	if filter == "" {
		posts, err = app.posts.All()
	} else {
		posts, err = app.posts.Filter(false, filter)
	}
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	tags, err := app.posts.AllUsedTags()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	data := app.newTemplateData(r)
	data.Posts = posts
	data.Tags = tags
	data.Flash = flash
	data.Filter = filter

	app.render(w, r, http.StatusOK, "post_list.tmpl", data)
}

func (app *application) repoList(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Repo List"))
}

func (app *application) draftList(w http.ResponseWriter, r *http.Request) {
	drafts, err := app.posts.AllDrafts()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	data := app.newTemplateData(r)
	data.Posts = drafts
	data.Flash = flash

	app.render(w, r, http.StatusOK, "draft_list.tmpl", data)
}

type postForm struct {
	Title               string `form:"title"`
	URL                 string `form:"url"`
	Tags                []models.Tag
	Body                string `form:"body"`
	IsDraft             bool   `form:"asDraft"`
	HeaderImageLocation string
	Description         string
	validator.Validator `form:"-"`
}

func (app *application) newPost(w http.ResponseWriter, r *http.Request) {
	form := postForm{}
	data := app.newTemplateData(r)
	data.NewPost = true
	data.Form = form

	tags, err := app.posts.AllTags()
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Tags = tags

	app.render(w, r, http.StatusOK, "add_edit_post.tmpl", data)
}

func (app *application) newPostSubmit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form postForm

	err = app.formDecoder.Decode(&form, r.PostForm)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 256), "title", "This field cannot be longer than 256 characters")
	form.CheckField(validator.NotBlank(form.Body), "body", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.URL), "url", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.URL, 200), "url", "This field cannot be longer than 200 characters")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		data.NewPost = true
		app.render(w, r, http.StatusUnprocessableEntity, "add_edit_post.tmpl", data)
		return
	}

	form.URL = url.PathEscape(form.URL)

	now := time.Now()
	_, err = app.posts.Insert(models.Post{
		Title:    form.Title,
		Body:     template.HTML(form.Body),
		Tags:     form.Tags,
		IsDraft:  form.IsDraft,
		Created:  now,
		Modified: now,
		URL:      form.URL,
		HeaderImage: models.Image{
			Location: form.HeaderImageLocation,
		},
		Description: form.Description,
	})

	if err != nil {
		if errors.Is(err, models.ErrDuplicateUrl) {
			form.AddFieldError("url", "URL already exists")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "add_edit_post.tmpl", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	if form.IsDraft {
		app.sessionManager.Put(r.Context(), "flash", "Saved")
		http.Redirect(w, r, "/blog/drafts", http.StatusSeeOther)
	} else {
		app.sessionManager.Put(r.Context(), "flash", "Published successfully!")
		http.Redirect(w, r, fmt.Sprintf("/blog/post/%s", form.URL), http.StatusSeeOther)
	}
}

func (app *application) editPost(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		app.notFound(w, r)
		return
	}

	post, err := app.posts.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	form := postForm{
		Title: post.Title,
		URL:   post.URL,
		Tags:  post.Tags,
		Body:  string(post.Body),
		HeaderImageLocation: post.HeaderImage.Location,
		Description: post.Description,
	}

	data := app.newTemplateData(r)
	data.Post = post
	data.Form = form
	data.NewPost = false

	tags, err := app.posts.AllTags()
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	data.Tags = tags

	app.render(w, r, http.StatusOK, "add_edit_post.tmpl", data)
}

func (app *application) editPostSubmit(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		app.notFound(w, r)
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form postForm

	err = app.formDecoder.Decode(&form, r.PostForm)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Title), "title", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.Title, 256), "title", "This field cannot be longer than 256 characters")
	form.CheckField(validator.NotBlank(form.Body), "body", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.URL), "url", "This field cannot be blank")
	form.CheckField(validator.MaxChars(form.URL, 200), "url", "This field cannot be longer than 200 characters")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		data.NewPost = false
		app.render(w, r, http.StatusUnprocessableEntity, "add_edit_post.tmpl", data)
		return
	}

	p, err := app.posts.Get(id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	now := time.Now()
	// If we are publishing a draft
	if !form.IsDraft && p.IsDraft {
		// I should probably create a new column called "published", but this
		// is fine for now
		p.Created = now
	}

	p.Title = form.Title
	p.Body = template.HTML(form.Body)
	p.IsDraft = form.IsDraft
	p.URL = url.PathEscape(form.URL)
	p.Tags = form.Tags
	p.Modified = now
	p.HeaderImage = models.Image{
		Location: form.HeaderImageLocation,
	}
	p.Description = form.Description
	
	err = app.posts.Update(p)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateUrl) {
			form.AddFieldError("url", "URL already exists")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "add_edit_post.tmpl", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	if form.IsDraft {
		app.sessionManager.Put(r.Context(), "flash", "Saved")
		http.Redirect(w, r, "/blog/drafts", http.StatusSeeOther)
	} else {
		app.sessionManager.Put(r.Context(), "flash", "Published successfully!")
		http.Redirect(w, r, fmt.Sprintf("/blog/post/%s", p.URL), http.StatusSeeOther)
	}
}

type loginForm struct {
	Username string
	Password string
	validator.Validator
}

func (app *application) login(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = loginForm{}
	app.render(w, r, http.StatusOK, "login.tmpl", data)
}

func (app *application) loginSubmit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := loginForm{
		Username: r.PostForm.Get("username"),
		Password: r.PostForm.Get("password"),
	}

	form.CheckField(validator.NotBlank(form.Username), "username", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Password), "password", "Thid field cannot be blank")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	id, err := app.users.Authenticate(form.Username, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)
	redirect := app.sessionManager.PopString(r.Context(), "redirectPathAfterLogin")
	if redirect == "" {
		redirect = "/"
	}

	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (app *application) logoutSubmit(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type uploadForm struct {
	validator.Validator
}

func (app *application) upload(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = uploadForm{}

	app.render(w, r, http.StatusOK, "upload.tmpl", data)
}

func (app *application) uploadPost(w http.ResponseWriter, r *http.Request) {
	form := uploadForm{}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		form.AddNonFieldError("Error Uploading Image")

		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
		return
	}

	files := r.MultipartForm.File["file"]

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		defer file.Close()

		buff := make([]byte, 512)
		_, err = file.Read(buff)
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		err = os.MkdirAll("./uploads", os.ModePerm)
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		f, err := os.Create(fmt.Sprintf("./uploads/%s", fileHeader.Filename))
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}

		defer f.Close()

		_, err = io.Copy(f, file)
		if err != nil {
			form.AddNonFieldError("Error Uploading Image")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusInternalServerError, "upload.tmpl", data)
			return
		}
	}

	data := app.newTemplateData(r)
	data.Form = form
	data.Flash = "Image uploaded successfully!"
	app.render(w, r, http.StatusOK, "upload.tmpl", data)
}

func (app *application) imagePicker(w http.ResponseWriter, r *http.Request) {
	files, err := os.Open("./uploads/")
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	defer files.Close()

	fileNames, err := files.Readdirnames(0)

	data := app.newTemplateData(r)
	data.FileNames = fileNames

	app.render(w, r, http.StatusOK, "imagepicker.tmpl", data)
}

func (app *application) headerImagePicker(w http.ResponseWriter, r *http.Request) {
	uploads, err := os.Open("./uploads/")
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	defer uploads.Close()

	fileNames, err := uploads.Readdirnames(0)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var finalFileNames []string
	for _, n := range fileNames {
		path := "./uploads/" + n
		bytes, err := os.ReadFile(path)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		mime := http.DetectContentType(bytes)
		if mime != "image/jpeg" && mime != "image/png" {
			continue
		}

		f, err := os.Open(path)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		defer f.Close()

		i, _, err := image.DecodeConfig(f)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		if i.Width != 1200 || i.Height != 630 {
			continue
		}

		finalFileNames = append(finalFileNames, n)
	}

	data := app.newTemplateData(r)
	app.logger.Info(data.CurrentPage)
	data.FileNames = finalFileNames
	app.render(w, r, http.StatusOK, "imagepicker.tmpl", data)
}
