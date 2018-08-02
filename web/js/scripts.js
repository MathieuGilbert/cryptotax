if (typeof app === "undefined" || app === null) app = {};
app.files = [];
app.trades = [];
app.newTrade = newTrade();
app.report = {
    type: "Holdings",
    currency: "",
    asOf: ""
};
app.reportItems = [];

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
