package org.devMandali.magnetPlay.controller;

import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.email.EmailServiceGrpc;
import org.devMandali.email.SendEmailRequest;
import org.devMandali.email.SendEmailResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.util.StringUtils;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/email")
public class SendEmailClientController {

    private final Logger logger = LoggerFactory.getLogger(SendEmailClientController.class);

    @GrpcClient("EmailService")
    private EmailServiceGrpc.EmailServiceBlockingStub emailServiceBlockingStub;

    public static record EmailSendRequest(
            String to,
            String from,
            String subject,
            String body
    ) {}

    @PostMapping
    public ResponseEntity<?> sendEmail(@RequestBody EmailSendRequest request) {
        try {
            logger.info("new request for email, subject: {}", request.subject);

            SendEmailRequest emailRequest = SendEmailRequest.newBuilder()
                    .setTo(request.to)
                    .setSubject(request.subject)
                    .setBody(request.body)
                    .setFrom(!StringUtils.hasText(request.from) ? "no.reply.magnetPlay@devmandali.com": request.from)
                    .build();

            logger.debug("Sending email to recipient, subject: {}", request.subject);

            SendEmailResponse emailResponse = emailServiceBlockingStub.sendEmail(emailRequest);

            if(!emailResponse.getSuccess()){
                logger.error("Something went wrong!!, {}", emailResponse.getError());
                return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(emailResponse.getError());
            }
            return ResponseEntity.ok(String.format("Email Sent Successfully!!, Your MessageId: %s", emailResponse.getMessageId()));
        } catch (Exception e) {
            logger.error("Failed to send email", e);
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body("Failed to send email. Please try again later.");
        }
    }
}
