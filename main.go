package main

/***********************
* Receipt Processor API - Kevin Adams
* Description:
* 	Implements a Receipt processor api as listed in the README
*	Includes on-save point calculation and input validation
*	Includes GET rq at /db to show structure of database
*	hosted at localhost:8080 for simplicity
***********************/
import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid"
)

type Receipt struct {
	Retailer     string `json:"retailer" binding:"required"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []struct {
		ShortDescription string `json:"shortDescription"`
		Price            string `json:"price"`
	}
	Total  string `json:"total"`
	Points int    `json:"-"`
}

type URI struct {
	ID string `uri:"id"`
}

func ReadReceipt(c *gin.Context) (Receipt, error) {
	var receipt Receipt

	err := json.NewDecoder(c.Request.Body).Decode(&receipt)
	if err != nil {
		//Invalid receipt passed
	} else {
		err := errors.New("")
		//Valid Decode, validate Receipt
		//Retailer is safe
		//Purchase date must be yyyy-mm-dd
		purchaseDate := receipt.PurchaseDate
		isdate, _ := regexp.Compile(`^[0-9]{4}-[0-1][1-9]-[0-2][0-9]`)
		if !isdate.MatchString(purchaseDate) {
			dateErr := errors.New("invalid Purchase Date Format ")
			err = errors.Join(err, dateErr)
		}
		//purchase time must be hh:mm
		purchaseTime := receipt.PurchaseTime
		istime, _ := regexp.Compile(`^[0-2][0-9]:[0-5][0-9]`)
		if !istime.MatchString(purchaseTime) {
			timeErr := errors.New("invalid Purchase Time Format ")
			err = errors.Join(err, timeErr)
		}
		//total must equal item prices sum
		//perform points check during this loop
		total, _ := strconv.ParseFloat(receipt.Total, 64)
		points := 0
		//50 points if the total is a round dollar amount with no cents.
		if math.Mod(total, 1) == 0 {
			points += 50
		}
		//25 points if the total is a multiple of `0.25`
		if math.Mod(total, 0.25) == 0 {
			points += 25
		}
		//One point for every alphanumeric character in the retailer name
		name := receipt.Retailer
		chars := strings.Split(name, "")
		j := 0
		for j < len(chars) {
			isalpha, _ := regexp.Compile(`^[a-zA-Z0-9]*$`)
			if isalpha.MatchString(chars[j]) {
				points += 1
			}
			j++
		}
		//5 points for every two items on the receipt.
		items := len(receipt.Items)
		points += (items / 2)
		//Loop through the items
		i := 0
		for i < len(receipt.Items) {
			itemPrice, _ := strconv.ParseFloat(receipt.Items[i].Price, 64)
			total -= itemPrice
			//calculate points for this item
			//If the trimmed length of the item description is a multiple of 3,/
			//multiply the price by `0.2` and round up to the nearest integer./
			//The result is the number of points earned.
			desc := receipt.Items[i].ShortDescription
			if len(desc)%3 == 0 {
				price, _ := strconv.ParseFloat(receipt.Items[i].Price, 64)
				price = price + (1 - math.Mod(price, 1))
				points += int(price)
			}
			//6 points if the day in the purchase date is odd.
			day := receipt.PurchaseDate[len(receipt.PurchaseDate)-2:]
			dayint, err := strconv.Atoi(day)
			if err != nil {
				//handle
			}
			if dayint%2 == 1 {
				points += 6
			}
			//10 points if the time of purchase is after 2:00pm and before 4:00pm.
			time := receipt.PurchaseTime[:2]
			timeint, err := strconv.Atoi(time)
			if err != nil {
				//handle
			}
			if 14 <= timeint && timeint < 16 {
				points += 10
			}
			i++
		}
		receipt.Points = points
		if total != 0 {
			totalErr := errors.New("total Price does not match item prices ")
			err = errors.Join(err, totalErr)
		}
		if len(err.Error()) == 0 {
			//if all checks pass, clear err for output
			err = nil
		}
	}
	return receipt, err
}

var db = make(map[string] /*shortuuid*/ Receipt)

func setupRouter() *gin.Engine {
	r := gin.New()
	r.SetTrustedProxies(nil)
	r.POST("/receipts/process", func(c *gin.Context) {
		receipt, err := ReadReceipt(c)
		if err == nil {
			uuid := shortuuid.New()

			db[uuid] = receipt
			c.JSON(200, gin.H{"uuid": uuid, "receipt": receipt})
		} else {
			c.JSON(400, gin.H{"msg": err})
		}
	})

	r.GET("/receipts/:id/points", func(c *gin.Context) {
		var uri URI
		var uuid string
		if err := c.ShouldBindUri(&uri); err != nil {
			c.JSON(400, gin.H{"msg": err.Error()})
			return
		}
		uuid = uri.ID
		receipt, ok := db[uuid]
		if !ok {
			c.JSON(400, gin.H{"msg": "That receipt does not exist.", "uuid": uuid})
			return
		}
		c.JSON(200, gin.H{"points": receipt.Points})
	})

	r.GET("/db", func(c *gin.Context) {
		c.JSON(200, gin.H{"database": db})
	})
	return r
}

func main() {
	//gin.SetMode(gin.ReleaseMode) //Disable for debug output
	r := setupRouter()

	r.Run(":8080")
}
