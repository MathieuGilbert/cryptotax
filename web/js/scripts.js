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

async function getLiveRates(from, tos, resolve, reject) {
    var rates = [];
    var url = "https://min-api.cryptocompare.com/data/price?fsym=" + from + "&tsyms=" + tos.join() + "&extraParams=cryptotax";

    await $.ajax({
        url: url,
        type: 'GET',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        if (data["Response"] !== "Error") {
            Object.keys(data).forEach(function(c) {
                rates.push({"currency" : c, "rate" : data[c]});
            });
        } else {
            return reject(JSON.stringify(data));
        }
        return resolve(rates);
    }).fail(function(xhr, status, error) {
        return reject(error);
    });
}

async function getRate(from, to, ts) {
    // return from cache
    var rate = cachedRate(from, to, ts);
    if (rate > -1) {
        return rate;
    }

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
            addRate(from, to, ts, rate);
        }
    }).fail(function(xhr, status, error) {
        console.log('failure: ' + error);
    });

    return rate;
}

function getRates(base, rates, ts, resolve, reject) {
    // currencies to look up
    var currs = rates.map(function(rate) {
        // check cache
        var r = cachedRate(base, rate.currency, ts);
        if (r > -1) {
            // set value from cache
            rate.rate = r;
            return;
        }
        // not cached, include it
        return rate.currency;
    });

    if (currs.length === 0) {
        return resolve(rates);
    }

    var url = "https://min-api.cryptocompare.com/data/pricehistorical?fsym=" + base + "&tsyms=" + currs.join() + "&ts=" + ts + "&extraParams=cryptotax&calculationType=MidHighLow"

    $.ajax({
        url: url,
        type: 'GET',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 10000,
        tryCount: 0,
        retryLimit: 5
    }).done(function(data) {
        if (data.Response !== "Error") {
            var resp = data[base];
            Object.keys(resp).forEach(function(currency) {
                addRate(base, currency, ts, resp[currency]);
                rates.map(function(rate) {
                    if (rate.currency === currency) {
                        rate.rate = String(resp[currency]);
                    }
                });
            });

            return resolve(rates);
        } else {
            // {"Response":"Error","Message":"Rate limit excedeed!","Type":99,"Aggregated":false,"Data":[],"YourCalls":{"hour":{"Histo":1661},"minute":{"Histo":364},"second":{"Histo":1}},"MaxLimits":{"Hour":8000,"Minute":300,"Second":15}}
            if (data.Type === 99) {
                var ms = data.YourCalls.second.Histo;
                var mm = data.YourCalls.minute.Histo;
                var mh = data.YourCalls.hour.Histo;
                var ls = data.MaxLimits.Second;
                var lm = data.MaxLimits.Minute;
                var lh = data.MaxLimits.Hour;

                var delay = 0;
                if (ms > ls) { delay = 1000; }
                if (mm > lm) { delay = 1000 * 60; }
                if (mh > lh) { delay = 1000 * 60 * 60; } // who's going to wait an hour?

                this.tryCount++;
                if (this.tryCount <= this.retryLimit) {
                    var self = this;
                    window.setTimeout(function() {
                        return; $.ajax(self);
                    }, delay);
                }
            } else {
                return reject(JSON.stringify(data));
            }
        }
    }).fail(function(xhr, status, error) {
        return reject(error);
    });
}

function addRate(from, to, ts, rate) {
    if (cachedRate(from, to, ts) == -1) {
        app.rates.push({from: from, to: to, ts: ts, rate: rate});
    }
}

function cachedRate(from, to, ts) {
    for (var i = 0; i < app.rates.length; i++) {
        var r = app.rates[i];
        if (r.ts === ts && r.from === from && r.to === to) {
            return r.rate;
        }
    }
    return -1;
}
