$(document).ready(function() {
    Vue.component('file-manager', {
        data() {
            return {
                files: app.files,
                trades: app.trades
            }
        },
        methods: {
            remove: function(e, file) {
                var fi = app.files.findIndex(f => f.id === file.id);
                if (fi > -1) {
                    app.files.splice(fi, 1);
                }
            },
            upload: function(e, file) {
                var s = e.currentTarget;
                if (s.selectedIndex > 0) {
                    uploadFile(file.id, s.value);
                }
            },
            confirmDelete: function(e, file) {
                var fi = app.files.findIndex(f => f.id === file.id);
                if (fi > -1) {
                    deleteFile(file, fi);
                    toggleDelete(e);
                }
            },
            keepFile: function(e) {
                toggleDelete(e);
            },
            wantDelete: function(e) {
                toggleDelete(e);
            },
            viewTrades: function(e, file) {
                selectRow(e);
                app.trades.splice(0, app.trades.length);
                getFileTrades(file.id);
            },
            shortDate: function(date) {
                return formatDate(date);
            },
            longDate: function(date) {
                return formatDateLong(date);
            },
            icon: function(file) {
                var fa = "";
                var status = "";

                switch (file.state) {
                case "addfailed":
                    fa = "fa-exclamation";
                    status = "red-progress";
                    break;
                case "added":
                    fa = "fa-question";
                    status = file.success ? "green-progress" : "red-progress";
                    break;
                case "uploading":
                     fa = "fa-spinner fa-spin";
                     status = "green-progress";
                     break;
                case "uploaded":
                     fa = "fa-check";
                     status = "green-progress";
                     break;
                case "deleting":
                     fa = "fa-spinner fa-spin";
                     status = "red-progress";
                     break;
                case "deletefailed":
                     fa = "fa-exclamation";
                     status = "red-progress";
                     break;
                }
                return status + " " + fa;
            }
        }
    });

    new Vue({
        delimiters: ['${', '}'],
        el: '#fm'
    });

    // load chosen files into grid
    $(':file').on('change', function() {
        // add valid files
        $.each(this.files, function(i, file) {
            var message;
            var success = true;

            if (file.size == 0) {
                message = "Empty file.";
                success = false;
            }
            if (file.size > 1 * 2**20) {
                message = "Max size 1 MB.";
                success = false;
            }
            if (file.type != "text/csv") {
                message = "Invalid CSV file.";
                success = false;
            }

            var f = {
                "id": generateUUID(),
                "name": file.name,
                "bytes": "",
                "state": success ? "added" : "addfailed",
                "exchange": "",
                "message": message,
                "success": success
            };

            if (!success) {
                app.files.push(f);
                return true;
            }

            // get byte array of file
            var callback = function(bytes) {
                // set values
                f.bytes = btoa(bytes);
                // store locally
                app.files.push(f);
            }
            var fr = new FileReader();
            fr.onload = function() {
                var array = new Uint8Array(fr.result);
                //var fileText = String.fromCharCode.apply(null, array);
                var fileBytes = JSON.stringify(Array.from(array));
                callback(fileBytes);
            }
            fr.readAsArrayBuffer(file);
        });

        // clear the processed file input
        var fileInput = $("form#file-upload").find("input[type='file']")[0];
        try {
            fileInput.value = null;
        } catch(ex) { }
        if (fileInput.value) {
            fileInput.parentNode.replaceChild(fileInput.cloneNode(true), fileInput);
        }
    });
});

function uploadFile(id, exchange) {
    // get the index of file in app.files
    var fi = app.files.findIndex(f => f.id === id);
    var file = app.files[fi];

    if (file.state !== "added") {
        return;
    }
    file.state = "uploading";
    file.success = true;
    file.message = "";

    var data = JSON.stringify({
        fileBytes: file.bytes,
        exchange: exchange,
        fileName: file.name,
        CSRFToken: $('input[name="csrf_token"]').val()
    });

    $.ajax({
        url: '/upload',
        type: 'POST',
        data: data,
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        // update from server
        file.id = data.fileId;
        file.date = data.date;
        file.exchange = data.exchange;
        file.message = data.message;
        file.success = data.success;
        file.state = data.success ? "uploaded" : "added";
    }).fail(function(e) {
        file.state = "added";
        file.success = false;
        file.message = "Failed to upload file.";
    });
}

function deleteFile(file, index) {
    if (file.state !== "uploaded") {
        return;
    }
    file.state = "deleting";
    file.success = true;
    file.message = "";

    var url = '/file?id=' + file.id + '&csrf_token=' + $('input[name="csrf_token"]').val();

    $.ajax({
        url: url,
        type: 'DELETE',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        app.files.splice(index, 1);
    }).fail(function(e) {
        file.state = "deletefailed";
        file.success = false;
        file.message = "Failed to delete file.";
    });
}

function getFileTrades(fid) {
    $.ajax({
        url: '/filetrades?id=' + fid,
        type: 'GET',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        for (var i = 0; i < data.trades.length; i++) {
            app.trades.push(data.trades[i]);
        }
    }).fail(function(e) {
        console.log('failure: ' + e);
    });
}

function selectRow(e) {
    $(".is-selected").removeClass("is-selected");
    var row = $(e.currentTarget).closest("tr");
    row.addClass("is-selected");
}

function toggleDelete(e) {
    var row = $(e.currentTarget).closest("tr")
    row.find(".delete-button").toggleClass("hidden");
    row.find(".view-button").toggleClass("hidden");
    row.find(".confirm-button").toggleClass("hidden");
    row.find(".keep-button").toggleClass("hidden");
}

function generateUUID() {
    var d = new Date().getTime();
    if (typeof performance !== 'undefined' && typeof performance.now === 'function'){
        d += performance.now();
    }
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
        var r = (d + Math.random() * 16) % 16 | 0;
        d = Math.floor(d / 16);
        return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
    });
}
