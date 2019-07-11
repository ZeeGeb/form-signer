package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"log"
   "io/ioutil"
	"bytes"
	 "encoding/json"
	"net/url"


	"github.com/jung-kurt/gofpdf"
	 "google.golang.org/appengine"
)

const assetDir = "assets/"


func main() {

	fs := http.FileServer(http.Dir(assetDir))
	 http.Handle("/", fs)
	http.HandleFunc("/api/render-pdf", renderPdf)

	 appengine.Main()

	//http.ListenAndServe(":8080", nil)
		http.HandleFunc("/serving",Serving)
		http.HandleFunc("/Logout",Logout)

    err := http.ListenAndServeTLS(":443", "server.crt", "server.key", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}

func Serving(w http.ResponseWriter,r *http.Request){
	myUrl,ok:=r.URL.Query()["code"]
	if !ok ||len(myUrl[0])<1{
		log.Println("Url Param 'access_token' is missing")
		return
	}
	myCode:=myUrl[0]
	clientS:="wncqpTUNK97$jqTRR766:-*"

	reqUrl:="https://login.live.com/oauth20_token.srf"
	reqBody:=url.Values{}
	reqBody.Set("client_id","9976be9f-59e7-4287-b7ff-8c2ba439523c")
	reqBody.Add("redirect_uri","https://localhost/serving")
	reqBody.Add("client_secret",clientS)
	reqBody.Add("code",myCode)
	reqBody.Add("grant_type","authorization_code")
	rb:=reqBody.Encode()
	byteRb:=[]byte(rb)

	reque,err:= http.NewRequest("POST",reqUrl,bytes.NewBuffer(byteRb))
 	// reque.Header.Add("Authorization","Bearer "+token)
	reque.Header.Set("Content-Type","application/x-www-form-urlencoded")
	client:=&http.Client{}
	respo,err:=client.Do(reque)


	if err != nil {
	 	log.Fatalln(err)
	}
	defer respo.Body.Close()
	body,err:=ioutil.ReadAll(respo.Body)
	if err !=nil{
		log.Fatalln(err)
	}
	log.Println(string(body))

	type Data struct{
	Token_type string `json:"token_type"`
	Expires_in int `json:"expires_in"`
	Scope string `json:"scope"`
	Access_token string `json:"access_token"`
	User_id string `json:"user_id"`
	Refresh_token string `json:"refresh_token"`
	}

	Token:=Data{}
	err=json.Unmarshal([]byte(string(body)),&Token)
	if err!=nil{
		fmt.Println("error: ",err)
	}
	myAccess_token:=Token.Access_token
	cookie:=http.Cookie{
	Name: "access_token",
	Value: myAccess_token,

	}
	reque,err= http.NewRequest("GET","https://graph.microsoft.com/v1.0/me/drive/root/children",nil)
	reque.Header.Add("Authorization","Bearer "+myAccess_token)
	reque.Header.Add("Content-type","application/json")

	if err!=nil{
		log.Fatalln(err)
	}

	respo,err=client.Do(reque)
	body,err=ioutil.ReadAll(respo.Body)
		if err !=nil{
			log.Fatalln(err)
		}
		log.Println(string(body))
	http.SetCookie(w,&cookie)

	http.Redirect(w,r,"https://localhost/new-patient.html",http.StatusSeeOther)

}

func Logout(w http.ResponseWriter,r *http.Request){
	fmt.Fprint(w,"<a href='https://login.live.com/oauth20_logout.srf?client_id=9976be9f-59e7-4287-b7ff-8c2ba439523c&redirect_uri=https://localhost/FirstPage.html'>Saved,Logout</a>")
}

func renderPdf(w http.ResponseWriter, r *http.Request) {

	c,error:=r.Cookie("access_token")
	if error!=nil{
		log.Println("Error in reading cookie : "+error.Error())
	}
		value:=c.Value
		log.Println("cookie has :"+value)

	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("could not parse form data: %v", err), http.StatusBadRequest)
		return
	}
	htmlName := assetDir[:len(assetDir)-1] + r.FormValue("_referrer")
	content, err := getPrintableContent(htmlName)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not get printable content for %v: %v", htmlName, err), http.StatusBadRequest)
		return
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 16)
	printParagraphs(content, pdf)
	printDate(pdf)
	printForm(content, r, pdf)
	printSignature(r, pdf)
	w.Header().Set("Content-Type", "application/pdf")
	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not create pdf: %v", err), http.StatusInternalServerError)
		return
	}
	client :=&http.Client{}
	reque,err:= http.NewRequest("PUT","https://graph.microsoft.com/v1.0/me/drive/root:/Files/SavedPdf:/content",&buf)
	reque.Header.Add("Authorization","Bearer "+value)

	reque.Header.Add("Content-type","/pdf")
	respo,err:=client.Do(reque)

	if err != nil {
		 log.Fatalln(err)
	}else{

		http.Redirect(w,r,"https://localhost/Logout",http.StatusSeeOther)
	}
	defer respo.Body.Close()
	body,err:=ioutil.ReadAll(respo.Body)
	if err !=nil{
		log.Fatalln(err)
	}
	log.Println(string(body))
}

func printDate(pdf *gofpdf.Fpdf) {
	pdf.Cell(40, 10, fmt.Sprintf("Date: %s", time.Now().Local().Format("Mon Jan 2 2006")))
	pdf.Ln(-1)
}

func printParagraphs(content *printable, pdf *gofpdf.Fpdf) {
	_, lineHt := pdf.GetFontSize()
	html := pdf.HTMLBasicNew()
	for _, par := range content.paragraphs {
		html.Write(lineHt, par)
		// Line break
		pdf.Ln(lineHt)
		pdf.Ln(lineHt)
	}
}

func printForm(content *printable, r *http.Request, pdf *gofpdf.Fpdf) {
	for k, v := range r.Form {
		if !strings.HasPrefix(k, "_") {
			for _, str := range v {
				pdf.Cell(40, 10, fmt.Sprintf("%s: %s", k, str))
				pdf.Ln(-1)
			}
		}
	}
}

func printSignature(r *http.Request, pdf *gofpdf.Fpdf) {
	svg := r.FormValue("_sigval")
	sig, err := gofpdf.SVGBasicParse([]byte(svg))
	if err == nil {
		scale := 100 / sig.Wd
		scaleY := 30 / sig.Ht
		if scale > scaleY {
			scale = scaleY
		}
		pdf.SetLineCapStyle("round")
		pdf.SetLineWidth(0.25)
		pdf.SetY(pdf.GetY() + 10)
		pdf.SVGBasicWrite(&sig, scale)
	} else {
		pdf.SetError(err)
	}
}
