package main

import (

	//"fyne.io/fyne/v2/app"

	"errors"
	"net"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {

	myApp := app.New()
	mainWindow := myApp.NewWindow("ZPL Sender")
	mainWindow.Resize(fyne.NewSize(600, 400))
	// TODO mettre une taille fixe et corriger les bugs de redimensionnement
	//mainWindow.SetFixedSize(true)

	ipEntry := widget.NewEntry()
	ipEntry.SetPlaceHolder("0.0.0.0")
	ipEntry.Validator = validation.NewRegexp(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`, "This is not a valid ip address")

	portEntry := newNumericalEntry()
	portEntry.SetPlaceHolder("Value between 0 and 65535")
	portEntry.Text = "6101"

	zplEntry := widget.NewMultiLineEntry()
	zplEntry.SetPlaceHolder("ZPL Commands must start with ^XA and end with ^XZ")
	zplEntry.SetMinRowsVisible(20)

	zplForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Printer ip address", Widget: ipEntry, HintText: "Printer ip address"},
			{Text: "Printer port", Widget: portEntry, HintText: "Printer port"},
			{Text: "Zpl commands", Widget: zplEntry, HintText: "ZPL commands"},
		},
		OnSubmit: func() {
			portInt, _ := strconv.ParseInt(portEntry.Text, 10, 64)
			if portInt < 0 || portInt > 65535 {
				err := errors.New("invalid port")
				dialog.ShowError(err, mainWindow)
				return
			}

			sendZplToPrinter(&mainWindow, ipEntry.Text, portEntry.Text, zplEntry.Text)
			//fmt.Println("Form: ", ipEntry.Text, portEntry.Text, zplEntry.Text)
		},
	}
	zplForm.SubmitText = "Send Zpl code to printer"

	mainWindow.SetContent(container.NewVBox(zplForm))
	mainWindow.ShowAndRun()
}

func sendZplToPrinter(window *fyne.Window, ip string, port string, zplCommands string) {
	conn, errConn := net.Dial("tcp", ip+":"+port)
	if errConn != nil {
		err := errors.New("impossible de se connecter à l'imprimante, vérifier l'adresse ip et le port")
		dialog.ShowError(err, *window)
		return
	}
	defer conn.Close()

	_, errSendZPL := conn.Write([]byte(zplCommands))
	if errSendZPL != nil {
		err := errors.New("erreur lors de l'envoi du ZPL à l'imprimante")
		dialog.ShowError(err, *window)
		return
	}

	dialog.ShowInformation("ZPL", "ZPL commands has been send to printer", *window)
}
