function InvokeFileUpload(file_data , file_path)
{
    let file_status_panel = document.createElement("div");
    file_status_panel.className = "file_upload_status";

    let panel_style_section = document.createElement("div");
    panel_style_section.className = "style_section";

    panel_style_section.appendChild(document.createElement("div"));
    panel_style_section.appendChild(document.createElement("div"));
    panel_style_section.appendChild(document.createElement("div"));

    let style_rotation_section = document.createElement("div");
    style_rotation_section.className = "rot";

    for(let i = 0 ; i < 5; ++i)
    {
        style_rotation_section.appendChild(document.createElement("div"));
    }

    panel_style_section.appendChild(style_rotation_section);

    file_status_panel.appendChild(panel_style_section);

    let file_name = document.createElement("h3");
    file_name.innerText = file_data.name;

    file_status_panel.appendChild(file_name);

    let progress_bar = document.createElement("div");
    progress_bar.className = "progress";

    let progress_bar_fill = document.createElement("div");
    progress_bar_fill.className = "progress_bar";

    progress_bar.appendChild(progress_bar_fill);

    file_status_panel.appendChild(progress_bar);

    let xhr = new XMLHttpRequest();
    xhr.open("POST", "/upload_file", true);
    xhr.upload.onprogress = function(event)
    {
        if(event.lengthComputable)
        {
            let progress = (event.loaded / event.total) * 100;
            progress_bar_fill.style.width = progress + "%";
        }
    };

    xhr.setRequestHeader("-file-name", (file_path.length > 0 ? file_path + "/" : "") + file_data.name);
    xhr.send(file_data);

    return file_status_panel;
}

function CreateFileDetailsPanel(thumbnail_img , name , size)
{
    let file_panel_container = document.createElement("div");
    file_panel_container.className = "file_panel_container";

    let content = document.createElement("div");
    content.className = "content";

    let style_section = document.createElement("div");
    style_section.className = "style_section";
    
    style_section.appendChild(document.createElement("div"));
    style_section.appendChild(document.createElement("div"));

    let details = document.createElement("div");
    details.className = "details";

    let thumbnail = document.createElement("img");
    thumbnail.src = thumbnail_img;
    thumbnail.className = "thumbnail";

    let file_name = document.createElement("h2");
    file_name.innerText = name;

    let file_size = document.createElement("h3");
    file_size.innerText = size;

    details.appendChild(thumbnail);
    details.appendChild(file_name);
    details.appendChild(file_size);
    
    content.appendChild(style_section);
    content.appendChild(details);
    file_panel_container.appendChild(content);

    return file_panel_container;
}