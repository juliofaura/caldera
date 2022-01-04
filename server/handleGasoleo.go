package server

import (
	"fmt"
	"math"
	"net/http"
	"os"

	"github.com/juliofaura/caldera/data"
	oildata "github.com/juliofaura/oilmeter/data"
	"github.com/juliofaura/oilmeter/files"
	"github.com/juliofaura/webutil"
	chart "github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const (
	timeForGraph = 61 * 24 * 60 * 60
)

func HandleGasoleo(w http.ResponseWriter, req *http.Request) {
	data.M.Lock()
	defer data.M.Unlock()

	// webutil.PushAlertf(w, req, webutil.ALERT_SUCCESS, "Success!")
	// webutil.Reload(w, req, "/")

	datums, err := files.ReadDataFile(files.DataFile)
	if err != nil {
		webutil.PushAlertf(w, req, webutil.ALERT_DANGER, "Error leyendo los datos del gasoleo")
		webutil.Reload(w, req, "/caldera")
		return
	}

	// avgFile, err := os.Open(files.AverageFile)
	// if err != nil {
	// 	return
	// }
	// defer avgFile.Close()

	// n, err := fmt.Fscanln(avgFile, &avg)
	// if n == 0 || err != nil {
	// 	webutil.PushAlertf(w, req, webutil.ALERT_DANGER, "Error leyendo los datos del consumo medio de gasoleo")
	// 	webutil.Reload(w, req, "/caldera")
	// 	return
	// }

	var XValues []float64
	var YValues []float64

	var histData [12]float64
	var histTime [12]int64

	for i, v := range datums {
		if i > 0 {
			thisDif := datums[i-1].Liters - v.Liters
			thisTime := v.Timestamp - datums[i-1].Timestamp
			if math.Abs(thisDif) < oildata.NewGasThreshold {
				histData[v.Month-1] += thisDif
				histTime[v.Month-1] += thisTime
			}
		}
		if v.Timestamp < datums[len(datums)-1].Timestamp-timeForGraph {
			continue
		}
		XValues = append(XValues, float64(v.Timestamp))
		YValues = append(YValues, v.Liters)
	}

	histStyle := chart.Style{
		FillColor:   drawing.ColorFromHex("FFFFFF"),
		StrokeColor: drawing.ColorFromHex("33AAAA"),
		StrokeWidth: 2,
	}

	histAvgs := []chart.Value{
		{Value: 1, Style: histStyle, Label: "Ene"},
		{Value: 0, Style: histStyle, Label: "Feb"},
		{Value: 0, Style: histStyle, Label: "Mar"},
		{Value: 0, Style: histStyle, Label: "Abr"},
		{Value: 0, Style: histStyle, Label: "May"},
		{Value: 0, Style: histStyle, Label: "Jun"},
		{Value: 0, Style: histStyle, Label: "Jul"},
		{Value: 0, Style: histStyle, Label: "Ago"},
		{Value: 0, Style: histStyle, Label: "Sep"},
		{Value: 0, Style: histStyle, Label: "Oct"},
		{Value: 0, Style: histStyle, Label: "Nov"},
		{Value: 0, Style: histStyle, Label: "Dec"},
		// {Value: 1, Label: "!!"},
	}

	for i, v := range histTime {
		if v > 0 {
			if histData[i] > 0 {
				histAvgs[i].Value = (histData[i] / float64(v)) * 24 * 60 * 60
			}
		}
	}

	LastX := XValues[len(XValues)-1]
	LastY := YValues[len(YValues)-1]

	var labelColor drawing.Color

	if LastY > oildata.AmountGood {
		labelColor = chart.ColorGreen
	} else if LastY > oildata.AmountDangerous {
		labelColor = chart.ColorYellow
	} else {
		labelColor = chart.ColorRed
	}

	var average = 0.0
	firstPointForAverage, endingPointForAverage := datums[len(datums)-1], datums[len(datums)-1]
	var bigChanges = 0.0
	for i := len(datums) - 2; i >= 0; i-- {
		if math.Abs(datums[i].Liters-datums[i+1].Liters) > oildata.NewGasThreshold {
			bigChanges += datums[i+1].Liters - datums[i].Liters
		}
		firstPointForAverage = datums[i]
		if endingPointForAverage.Timestamp-firstPointForAverage.Timestamp >= int64(oildata.TimeForAverage) {
			break
		}
	}

	if firstPointForAverage.Timestamp != endingPointForAverage.Timestamp {
		average = -float64(endingPointForAverage.Liters-firstPointForAverage.Liters-bigChanges) / (float64(endingPointForAverage.Timestamp-firstPointForAverage.Timestamp) / (24 * 60 * 60))
	}

	graph1 := chart.Chart{
		XAxis: chart.XAxis{
			TickPosition: chart.TickPositionBetweenTicks,
			ValueFormatter: func(v interface{}) string {
				typed := v.(float64) * 1e9
				typedDate := chart.TimeFromFloat64(typed)
				return fmt.Sprintf("%d/%d/%d", typedDate.Month(), typedDate.Day(), typedDate.Year())
			},
			Style: chart.Style{
				TextRotationDegrees: 45,
			},
		},
		YAxis: chart.YAxis{
			ValueFormatter: func(v interface{}) string {
				return fmt.Sprintf("%.1f", v.(float64))
			},
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Style: chart.Style{
					StrokeColor: chart.GetDefaultColor(0).WithAlpha(64),
					FillColor:   chart.GetDefaultColor(0).WithAlpha(64),
					StrokeWidth: 5,
					DotWidth:    4,
				},
				XValues: XValues,
				YValues: YValues,
			},
			chart.AnnotationSeries{
				Annotations: []chart.Value2{
					{
						XValue: LastX,
						YValue: LastY,
						Label:  fmt.Sprintf("%.1f", LastY),
						Style: chart.Style{
							StrokeWidth: 10,
							FontSize:    chart.StyleTextDefaults().FontSize,
							StrokeColor: labelColor,
						},
					},
				},
			},
		},
		Title: "Oil liters vs time (avg is " + fmt.Sprintf("%.2f", average) + " liters / day)",
	}

	f, _ := os.Create(RESOURCES_DIR + "gasoleo.png")
	defer f.Close()
	graph1.Render(chart.PNG, f)

	graph2 := chart.BarChart{
		Title: "Test Bar Chart",
		Background: chart.Style{
			Padding: chart.Box{
				Top: 40,
			},
		},
		Height:   512,
		BarWidth: 90,
		Bars:     histAvgs,
	}

	f, _ = os.Create(RESOURCES_DIR + "consumos.png")
	defer f.Close()
	graph2.Render(chart.PNG, f)

	passdata := map[string]interface{}{
		"liters":         fmt.Sprintf("%.1f", datums[len(datums)-1].Liters),
		"avg":            fmt.Sprintf("%.1f", average),
		"daysforaverage": oildata.TimeForAverage / (60 * 60 * 24),
		"datums":         datums,
	}
	webutil.PlaceHeader(w, req)
	templates.ExecuteTemplate(w, "gasoleo.html", passdata)
}
