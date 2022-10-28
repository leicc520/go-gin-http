package parse

import (
	"errors"
	"fmt"
	core "git.ziniao.com/webscraper/go-gin-http"
	"regexp"
	"testing"
)

func TestError(t *testing.T) {
	e := ParseError{}
	err := errors.New("demo test")
	e.Wrapped("demo", err)
	e.Wrapped("demov2", err)

	fmt.Println(e, e.IsEmpty())
}

func TestCssPath(t *testing.T) {
	str := `<div id="corePriceDisplay_desktop_feature_div" class="celwidget" data-feature-name="corePriceDisplay_desktop" data-csa-c-id="4ysw9w-og8yfs-mibj67-oztinb" data-cel-widget="corePriceDisplay_desktop_feature_div">
                                            <style type="text/css">
    .savingPriceOverride {
        color:#CC0C39!important;
        font-weight: 300!important;
    }
</style>

                       <div class="a-section a-spacing-none aok-align-center">       <span class="a-size-large a-color-price savingPriceOverride aok-align-center reinventPriceSavingsPercentageMargin savingsPercentage">-6%</span>         <span class="a-price aok-align-center reinventPricePriceToPayMargin priceToPay" data-a-size="xl" data-a-color="base"><span class="a-offscreen">$29.99</span><span aria-hidden="true"><span class="a-price-symbol">$</span><span class="a-price-whole">29<span class="a-price-decimal">.</span></span><span class="a-price-fraction">99</span></span></span>               </div>  <div class="a-section a-spacing-small aok-align-center"> <span> <span class="a-size-small a-color-secondary aok-align-center basisPrice">Was:         <span class="a-price a-text-price" data-a-size="s" data-a-strike="true" data-a-color="secondary"><span class="a-offscreen">$31.99</span><span aria-hidden="true">$31.99</span></span>     </span>  <span class="a-size-small aok-align-center basisPriceLegalMessage">                                 <a class="a-link-normal" href="https://www.amazon.com/gp/help/customer/display.html?nodeId=GQ6B6RH72AX8D2TD&amp;ref_=dp_hp">
                        <svg aria-hidden="true" class="reinventPrice_legalMessage_icon" role="img" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
                           <path d="M256,9C119,9,8,120.08,8,257S119,505,256,505,504,394,504,257,393,9,256,9Zm0,76.31A47.69,47.69,0,1,1,208.31,133,47.69,47.69,0,0,1,256,85.31Zm38.15,332.38a12.18,12.18,0,0,1-12.21,12H229.67a11.85,11.85,0,0,1-11.82-12V249.92a11.86,11.86,0,0,1,11.82-12h52.27a12.18,12.18,0,0,1,12.21,12Z"></path>
                        </svg>
                      </a>
                             <style type="text/css">
    .reinventPrice_legalMessage_icon {
        width: 12px;
        fill: #969696;
        vertical-align: middle;
        padding-bottom: 2px;
    }

    .reinventPrice_legalMessage_icon:hover {
        fill: #555555;
    }
</style>


  </span> </span> </div>                                    </div>`
	tt, err := NewQueryParse(str)
	fmt.Println(tt, err)
	astr, err := tt.InnerText("#corePriceDisplay_desktop_feature_div .priceToPay .a-offscreen")
	fmt.Println(astr, err)
}

func TestReg(t *testing.T) {
	ss := "[\\d]+"
	result := "11adsf123dfdfdf"
	if reg, err := regexp.Compile(ss); err == nil {
		arrStr := reg.FindAllString(result, -1)
		if len(arrStr) > 0 {
			fmt.Println(arrStr)
		}
	}
}

func TestQuery(t *testing.T) {
	str := `<div class="a-box-inner a-padding-medium"><!-- Detailed Seller Information -->
                <div class="a-row a-spacing-small"><h3>Detailed Seller Information</h3></div><div class="a-row a-spacing-none"><span class="a-text-bold">Business Name:
                            </span><span>Corso Bale.Inc</span></div><div class="a-row a-spacing-none"><span class="a-text-bold">Business Address:
                            </span></div><div class="a-row a-spacing-none indent-left"><span>5822 W Third ST #101</span></div><div class="a-row a-spacing-none indent-left"><span>Los Angeles</span></div><div class="a-row a-spacing-none indent-left"><span>CA</span></div><div class="a-row a-spacing-none indent-left"><span>90036</span></div><div class="a-row a-spacing-none indent-left"><span>US</span></div><!-- Detailed Seller Information -->
            </div>`
	tt, err := NewQueryParse(str)
	fmt.Println(tt, err)
	astr, err := tt.InnerTexts(".indent-left span")
	fmt.Println(astr, err)
	str = core.StripTags(str)
	fmt.Println(str)
}

func TestTable(t *testing.T) {
	str := `<table>
<tr><th></th><th class="a-text-right">30 days</th>
<th class="a-text-right">90 days</th>
<th class="a-text-right">12 months</th>
<th class="a-text-right">Lifetime</th></tr>

<tr><td class="a-nowrap" style="width:1px;">Positive</td> <td class="a-text-right"> <span class="a-color-success">63</span>% </td> <td class="a-text-right"> 
<span class="a-color-success">68</span>% </td> <td class="a-text-right"> <span class="a-color-success">75</span>% </td> <td class="a-text-right"> 
<span class="a-color-success">89</span>% </td></tr>
<tr><td class="a-nowrap" style="width:1px;">Neutral</td> <td class="a-text-right"> <span class="a-color-secondary">3</span>% </td> <td class="a-text-right"> 
<span class="a-color-secondary">1</span>% </td> <td class="a-text-right"> <span class="a-color-secondary">3</span>% </td> <td class="a-text-right">
<span class="a-color-secondary">2</span>% </td></tr><tr><td class="a-nowrap" style="width:1px;">Negative</td> <td class="a-text-right">
<span class="a-color-error">34</span>% </td> <td class="a-text-right"> <span class="a-color-error">31</span>% </td> <td class="a-text-right"> 
<span class="a-color-error">22</span>% </td> <td class="a-text-right"> <span class="a-color-error">9</span>% </td></tr>
<tr><td class="a-nowrap" style="width:1px;">Count</td><td class="a-text-right"><span>38</span></td><td class="a-text-right"><span>120</span>
</td><td class="a-text-right"><span>549</span></td><td class="a-text-right"><span>12,820</span></td>
</tr></table>`
	fmt.Println(str)

	tt, err := NewQueryParse(str)
	fmt.Println(tt, err)
	astr, err := tt.InnerTexts("table tr")
	fmt.Println(astr, err)

	expr := `//table[@id='feedback-summary-table']`
	ok, err := regexp.MatchString(`//table\[[^\]]+\]`, expr)
	fmt.Println(ok, err)
}
