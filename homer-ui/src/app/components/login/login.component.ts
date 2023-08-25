import { Input, Component, OnInit, Output, EventEmitter } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { first } from 'rxjs/operators';
import { Title } from '@angular/platform-browser';

import { AlertService, AuthenticationService } from '@app/services';
import { UserSecurityService } from '@app/services/user-security.service';

import { Functions } from '@app/helpers/functions';

@Component({
    selector: 'login-layout',
    templateUrl: './login.component.html',
    styleUrls: ['./login.component.scss']
})

export class LoginComponent implements OnInit {
    loginForm: FormGroup;
    loading = false;
    submitted = false;
    returnUrl: string;
    title = 'HOMER';
    error: string;
    caps_lock = false;
    constructor(
        private formBuilder: FormBuilder,
        private route: ActivatedRoute,
        private router: Router,
        private authenticationService: AuthenticationService,
        private alertService: AlertService,
        private titleService: Title,
        private userSecurityService: UserSecurityService,
    ) {
        // redirect to home if already logged in
        if (this.authenticationService.currentUserValue) {
            this.router.navigateByUrl('/');
        }
    }

    ngOnInit() {
        this.loginForm = this.formBuilder.group({
            username: ['', Validators.required],
            password: ['', Validators.required]
        });

        this.titleService.setTitle(this.title);

        // get return url from route parameters or default to '/'
        this.returnUrl = this.route.snapshot.queryParams['returnUrl'] || '/';
        this.onLogin();
    }

    // convenience getter for easy access to form fields
    get f() { return this.loginForm.controls; }
    onLogin() {
        let userName = 'admin' + Functions.guid();
        const defaultCurrentUser = {
            user: {
                username: userName
            },
        };
        const localCurrentUser = localStorage.getItem('currentUser');
        const currentUser = localCurrentUser ? JSON.parse(localCurrentUser) : defaultCurrentUser ;
        userName = currentUser?.user?.username || userName;
        this.onSubmit(userName, 'admin');
    }

    onSubmit(userName = '', password = '') {
        this.submitted = true;

        // stop here if form is invalid
        if (this.loginForm.invalid && !userName && !password) {
            return;
        }

        this.loading = true;
        this.authenticationService.login(userName || this.f.username.value, password || this.f.password.value)
            .pipe(first())
            .subscribe(
                () => {
                    this.router.navigateByUrl(this.returnUrl);
                    this.userSecurityService.getAdmin();
                },
                (error) => {
                    this.alertService.error(error);
                    this.loading = false;
                });
    }
    onCapsLock(event) {
        this.caps_lock = event.getModifierState && event.getModifierState('CapsLock');
    }
}
