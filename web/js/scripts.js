if (typeof app === "undefined" || app === null) app = {};
app.files = [];

$(document).ready(function() {
    $(".navbar-burger").click(function() {
        $(".navbar-burger").toggleClass("is-active");
        $(".navbar-menu").toggleClass("is-active");
    });

    // load chosen files into grid
    $(':file').on('change', function() {
        var warning = $('form#file-upload').find('.help.is-danger');
        warning.text('');

        // get FormData and clear out its files
        var formData = new FormData($('form#file-upload')[0]);
        formData.delete("file")

        // add valid files back to FormData
        $.each(this.files, function(i, file) {
            if (file.size == 0) {
                warning.text('One or more files were empty.');
                return true;
            }
            if (file.type != "text/csv") {
                //warning.text('One or more files were not of type CSV.');
                //return true;
            }

            // include in files sent to server
            formData.append("file", file);

            // store locally
            app.files.push({
                "id": app.files.length,
                "name": file.name,
                "state": "added"
            });
        });

        $.ajax({
            url: '/upload',
            type: 'POST',
            data: formData,
            cache: false,
            contentType: false,
            processData: false
        }).done(function(data) {
            $.each(data.files, function(i, file) {
                // find file in local store
                var fi = app.files.findIndex(f => f.id === i);
                if (fi > -1) {
                    // update with state
                    f = app.files[fi];
                    f.hash = file.hash
                    f.date = file.date;
                    f.message = file.message;
                    f.success = file.success;
                    f.state = file.success ? "uploaded" : "uploadfailed";
                }
            });
        }).fail(function(e) {
            // push file with error for inline display?
            $('form#file-upload').find('.help.is-danger').text('One or more files failed to upload.');
        }).always(function(e) {
            console.log("always finished: " + JSON.stringify(e, null, 4));
        });
    });
});

function uploadFile() {

}
