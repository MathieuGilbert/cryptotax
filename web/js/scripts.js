if (typeof app === "undefined" || app === null) app = {
    files: [],
    trades: [],
    newTrade: newTrade(),
    report: {
        type: "Holdings",
        currency: "",
        locale: navigator.language,
        asOf: ""
    },
    reportItems: [],
    rates: [],
};


$(document).ready(function() {
    // Bulma hamburger nav
    $(".navbar-burger").click(function() {
        $(".navbar-burger").toggleClass("is-active");
        $(".navbar-menu").toggleClass("is-active");
    });
});

function newTrade() {
    return {
        id: "",
        date: "",
        action: "BUY",
        amount: "",
        currency: "",
        baseAmount: "",
        baseCurrency: "",
        feeAmount: "",
        feeCurrency: "",
        error: ""
    };
}

function formatDate(date) {
    if (date === undefined || date === "") {
        return "";
    }
    var dt = new Date(date);
    var y = dt.getFullYear();
    var m = zeroPad(dt.getMonth() + 1);
    var d = zeroPad(dt.getDate());

    return y + '-' + m + '-' + d;
}

function formatDateLong(date) {
    if (date === undefined || date === "") {
        return "";
    }
    var dt = new Date(date);
    var y = dt.getFullYear();
    var m = zeroPad(dt.getMonth() + 1);
    var d = zeroPad(dt.getDate());
    var h = zeroPad(dt.getHours());
    var min = zeroPad(dt.getMinutes());

    return y + '-' + m + '-' + d + ' ' + h + ':' + min;
}

function zeroPad(s) {
    return ("00" + s).slice(-2);
}

async function getRate(from, to, ts) {
    var rate = cachedRate(from, to, ts);
    if (rate > -1) {
        return rate;
    }

    // can look up multiple values with this endpoint (numbers are a bit different):
    // https://min-api.cryptocompare.com/data/pricehistorical?fsym=CAD&tsyms=ETH,BTC,NEO&ts=1533330292&extraParams=cryptotax&calculationType=MidHighLow
    var url = "https://min-api.cryptocompare.com/data/dayAvg?fsym=" + from + "&tsym=" + to + "&toTs=" + ts + "&extraParams=cryptotax";

    await $.ajax({
        url: url,
        type: 'GET',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        if (data["Response"] !== "Error") {
            rate = data[to];
            app.rates.push({from: from, to: to, ts: ts, rate: rate});
        }
    }).fail(function(e) {
        console.log('failure: ' + e);
    });

    return rate;
}

function cachedRate(from, to, ts) {
    for (var i = 0; i < app.rates.length; i++) {
        var r = app.rates[i];
        if (r.from === from && r.to === to && r.ts === ts) {
            return r.rate;
        }
    }
    return -1;
}
