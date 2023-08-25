import { Component, Inject } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA} from '@angular/material/dialog';

@Component({
    selector: 'app-setting-result-widget-component',
    templateUrl: 'setting-result-widget.component.html',
    styleUrls: ['./setting-result-widget.component.scss']
})
export class SettingResultWidgetComponent {
    constructor(
        public dialogRef: MatDialogRef<SettingResultWidgetComponent>,
        @Inject(MAT_DIALOG_DATA) public data: any
    ) { }

    onNoClick(): void {
        this.dialogRef.close();
    }
}
