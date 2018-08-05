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
                }
            },
            deep: true
        }
    }
});

new Vue({
    delimiters: ['${', '}'],
    el: '#rv'
});

async function loadReport(report) {
    app.loadingReport = true;
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
            setError(data.error);
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

        $(document).ajaxStop(function() {
            if (app.loadingReport === true) {
                getComputedReport(report, items);
            }
            app.loadingReport = false;
        });

        Promise.all(promises).then(function() {
            console.log(JSON.stringify(items));
            //app.loadingReport = true;
    	}).catch(function(e) {
            console.log(e);
        });
    }).fail(function(xhr, status, error) {
        console.log(error);
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

    $.ajax({
        url: "/report",
        type: "POST",
        data: data,
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
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

        p.then((rates) => {
            data.items.map(item => {
                var amount = Number(item.amount);
                var curr = item.asset;
                var rate = rates.filter(r => r.currency === curr);
                if (rate.length > 0) {
                    var value = rate[0].rate === 0 ? 0 : amount / rate[0].rate;
                    var gain = value / item.acb - 1;

                    app.reportItems.push({
                        asset: item.asset,
                        amount: amount,
                        acb: item.acb,
                        value: value,
                        gain: gain
                    });
                }
            });
        });

    }).fail(function(xhr, status, error) {
        console.log("Error loading report");
    });
}

function setError(text) {
    $('.help.is-danger').text(text);
}
