package main

import (

	//"fyne.io/fyne/v2/app"

	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type printerConnection struct {
	ip             string
	port           string
	conn           *net.TCPConn
	window         *fyne.Window
	outputRecvData *widget.Entry
}

func (p *printerConnection) connectToPrinter() error {

	tcpAddr, err := net.ResolveTCPAddr("tcp", p.ip+": "+p.port)
	if err != nil {
		return err
	}
	conn, errConn := net.DialTCP("tcp", nil, tcpAddr)
	p.conn = conn
	if errConn != nil {
		err := errors.New("impossible de se connecter à l'imprimante, vérifier l'adresse ip et le port")
		dialog.ShowError(err, *p.window)
		return err
	}
	return nil
}

func (p *printerConnection) recvFromPrinter() {
	input := bufio.NewScanner(p.conn)
	for input.Scan() {
		slog.Debug(fmt.Sprintf("%s - << %q\n", p.conn.RemoteAddr(), input.Text()))
		fyne.Do(func() {
			p.outputRecvData.Append(input.Text() + "\n")
		})
	}
	if err := input.Err(); err != nil {
		// erreur sur le scan de la saisie
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			slog.Error(fmt.Sprintln("Erreur sur le buffer de reception d'un client ", err))
		}
	}
}

func (p *printerConnection) sendToPrinter(commands string) {
	_, errSendComands := fmt.Fprint(p.conn, commands)
	if errSendComands != nil {
		dialog.ShowError(errSendComands, *p.window)
	}

}

func main() {
	myApp := app.New()
	mainWindow := myApp.NewWindow("Commands Sender")
	mainWindow.Resize(fyne.NewSize(800, 600))
	// TODO mettre une taille fixe et corriger les bugs de redimensionnement
	//mainWindow.SetFixedSize(true)

	ipEntry := widget.NewEntry()
	ipEntry.SetPlaceHolder("0.0.0.0")
	ipEntry.Validator = validation.NewRegexp(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`, "This is not a valid ip address")

	portEntry := newNumericalEntry()
	portEntry.SetPlaceHolder("Value between 0 and 65535")
	portEntry.Text = "9100"

	commandsEntry := widget.NewMultiLineEntry()
	commandsEntry.SetPlaceHolder("Commands to send to printer, ZPL Commands must start with ^XA and end with ^XZ, SGD commands must end with \r\n")
	commandsEntry.SetMinRowsVisible(15)

	contentReturnLabel := widget.NewLabel("Returned values:")
	contentReturn := widget.NewMultiLineEntry()
	contentReturn.SetMinRowsVisible(10)
	contentReturn.Disable()

	connection := printerConnection{window: &mainWindow, outputRecvData: contentReturn}

	commandForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Printer Commands", Widget: commandsEntry, HintText: "ZPL or SGD commands"},
		},
		OnSubmit: func() {
			connection.sendToPrinter(commandsEntry.Text)
			dialog.ShowInformation("Command send", "Commands has been send to printer", mainWindow)
		},
	}
	commandForm.SubmitText = "Send Zpl code to printer"

	printerConnectForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Printer ip address", Widget: ipEntry, HintText: "Printer ip address"},
			{Text: "Printer port", Widget: portEntry, HintText: "Printer port"},
		},
		OnSubmit: func() {
			portInt, _ := strconv.ParseInt(portEntry.Text, 10, 64)
			if portInt < 0 || portInt > 65535 {
				err := errors.New("invalid port")
				dialog.ShowError(err, mainWindow)
				return
			}
			connection.ip = ipEntry.Text
			connection.port = portEntry.Text
			if err := connection.connectToPrinter(); err == nil {
				commandForm.Enable()
				go connection.recvFromPrinter()
			}
		},
	}
	printerConnectForm.SubmitText = "Connect to printer"

	mainWindow.SetContent(container.NewVBox(printerConnectForm, commandForm, contentReturnLabel, contentReturn))
	mainWindow.ShowAndRun()
}

func sendZplToPrinter(window *fyne.Window, outWidget *widget.Entry, ip string, port string, zplCommands string) {
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

	// récupère le paquet de retour
	for {
		input := bufio.NewScanner(conn)
		for input.Scan() {
			slog.Debug(fmt.Sprintf("%s - << %q\n", conn.RemoteAddr(), input.Text()))
			if len(input.Text()) > 0 {
				outWidget.Enable()
				outWidget.Append(input.Text() + "\n")
				outWidget.Disable()
				return
			}
		}
		if err := input.Err(); err != nil {
			// erreur sur le scan de la saisie
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				slog.Error(fmt.Sprintln("Erreur sur le buffer de reception d'un client ", err))
			}
		}
	}

}
