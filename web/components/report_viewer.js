Vue.component('report-viewer', {
    data() {
        return {
            report: app.report,
            items: app.reportItems,
            rates: app.rates
        }
    },
    methods: {
        setLocale: function(e) {
            switch ($(e.currentTarget).val()) {
            case "CAD":
                this.report.locale = "en-CA";
                break;
            }
        },
        currency: function(val) {
            var formatter = new Intl.NumberFormat(this.report.locale, {
                style: 'currency',
                currency: this.report.currency,
            });

            return formatter.format(val);
        },
        percent: function(val) {
            var formatter = new Intl.NumberFormat(this.report.locale, {
                style: 'percent',
                minimumFractionDigits: 2,
            });

            return formatter.format(val);
        }
    },
    watch: {
        report: {
            handler: function(report) {
                if (report.type !== "" && report.currency !== "" && report.asOf !== "") {
                    loadReport(report);
                    //.then(function(x) {
                    //    console.log("done");
                    //    console.log(x);
                    //    console.log(app.rates.length);
                    //});
                }
            },
            deep: true
        },
        //rates: {
        //    handler: function(rates) {
        //        console.log(rates.length);
        //    }
        //}
    }
});

new Vue({
    delimiters: ['${', '}'],
    el: '#rv'
});

async function loadReport(report) {
    app.reportItems.splice(0, app.reportItems.length);
    setError("");

    var url = '/rateRequest?type=' + report.type + '&currency=' + report.currency + '&asof=' + report.asOf;
    url += '&csrf_token=' + $('input[name="csrf_token"]').val();

    await $.ajax({
        url: url,
        type: 'GET',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 21000
    }).done(function(data) {
        if (data.error.length) {
            setError(data.error); // TODO: make this work with vue data
            return;
        }

        // [{"timestamp":1512266000,"rates":[{"currency":"ETH","rate":""},{"currency":"BTC","rate":""}]},{"timestamp":1515036507,"rates":[{"currency":"ETH","rate":""}]},
        var items = data.items;
        var promises = [];

        for (var i = 0; i < items.length; i++) {
            var req = items[i]; // {"timestamp":1512266000,"rates":[{"currency":"ETH","rate":""},{"currency":"BTC","rate":""}]}

    		promises.push(new Promise(function(resolve, reject) {
                getRates(app.report.currency, req.rates, req.timestamp, resolve, reject);
            }));
        }

        Promise.all(promises).then(function() {
            console.log(JSON.stringify(items));
            getComputedReport(report, items);
    	});
    }).fail(function(xhr, status, error) {
        console.log("Error loading report");
    });
}

// TODO: make separate endpoint so this can be POSTed
function getComputedReport(report, rates) {
    var data = JSON.stringify({
        type: report.type,
        currency: report.currency,
        asof: report.asOf,
        rates: rates,
        CSRFToken: $('input[name="csrf_token"]').val()
    });
    debugger;
    $.ajax({
        url: "/report",
        type: "POST",
        data: data,
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        debugger;
        if (data.error.length) {
            setError(data.error); // TODO: make this work with vue data
            return;
        }

        var currs = data.items.map(item => {
            return item.asset;
        });

        var p = new Promise(function(resolve, reject) {
            getLiveRates(app.report.currency, currs, resolve, reject);
        });

        p.then((rate) => {
            var amount = Number(item.amount)
            var value = amount * rate;
            var gain = value / amount - 1;

            app.reportItems.push({
                asset: item.asset,
                amount: amount,
                acb: item.acb,
                value: value,
                gain: gain
            });
        });

    }).fail(function(xhr, status, error) {
        console.log("Error loading report");
    });
}

function setError(text) {
    $('.help.is-danger').text(text);
}
