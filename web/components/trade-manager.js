Vue.component('trade-manager', {
    data() {
        return {
            trades: app.trades,
            newTrade: app.newTrade
        }
    },
    methods: {
        clearRow: function(e) {
            resetTrade(app.newTrade);
        },
        addTrade: function(e) {
            var t = app.newTrade;

            if (t.date === undefined || t.date.length !== 10) {
                t.error = "Invalid date.";
                return;
            }
            if (t.action === undefined || (t.action !== "BUY" && t.action !== "SELL")) {
                t.error = "Must be BUY or SELL.";
                return;
            }
            if (t.amount === undefined || t.amount.length === 0) {
                t.error = "Amount missing.";
                return;
            }
            if (t.currency === undefined || t.currency.length === 0) {
                t.error = "Currency missing.";
                return;
            }
            if (t.baseAmount === undefined || t.baseAmount.length === 0) {
                t.error = "For amount missing.";
                return;
            }
            if (t.baseCurrency === undefined || t.baseCurrency.length === 0) {
                t.error = "For currency missing.";
                return;
            }
            if (t.feeAmount === undefined || t.feeAmount.length === 0) {
                t.error = "Fee amount missing.";
                return;
            }
            if (t.feeCurrency === undefined || t.feeCurrency.length === 0) {
                t.error = "Fee currency missing.";
                return;
            }

            saveTrade(t);
        }
    }
});

new Vue({
    delimiters: ['${', '}'],
    el: '#tm'
});

function resetTrade(t) {
    t.id = ""
    t.date = ""
    t.action = "BUY"
    t.amount = ""
    t.currency = ""
    t.baseAmount = ""
    t.baseCurrency = ""
    t.feeAmount = ""
    t.feeCurrency = ""
    t.error = ""
}

function saveTrade(trade) {
    var data = JSON.stringify({
        trade: trade,
        CSRFToken: $('input[name="csrf_token"]').val()
    });

    $.ajax({
        url: '/trade',
        type: 'POST',
        data: data,
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        app.trades.push(data.trade);
        resetTrade(app.newTrade);
    }).fail(function(e) {
        app.newTrade.error = "Couldn't save trade.";
    });
}
