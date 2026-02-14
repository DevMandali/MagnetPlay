package org.devMandali.magnetPlay.email.service;

import io.grpc.stub.StreamObserver;
import net.devh.boot.grpc.server.service.GrpcService;
import org.devMandali.email.EmailServiceGrpc;
import org.devMandali.email.SendEmailRequest;
import org.devMandali.email.SendEmailResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.mail.SimpleMailMessage;
import org.springframework.mail.javamail.JavaMailSender;

import java.util.UUID;

@GrpcService
public class EmailServiceImpl extends EmailServiceGrpc.EmailServiceImplBase {

    private static final Logger logger = LoggerFactory.getLogger(EmailServiceImpl.class);
    private final JavaMailSender mailSender;

    public EmailServiceImpl(JavaMailSender mailSender) {
        this.mailSender = mailSender;
    }

    @Override
    public void sendEmail(SendEmailRequest request, StreamObserver<SendEmailResponse> responseObserver) {
        logger.info("Recieved request to send email to: {}", request.getTo());

        try {
            // Create and send email using Spring's mail sender
            SimpleMailMessage message = new SimpleMailMessage();
            message.setTo(request.getTo());
            message.setSubject(request.getSubject());
            message.setText(request.getBody());

            if(!request.getFrom().isEmpty()) {
                message.setFrom(request.getFrom());
            }


            mailSender.send(message);

            // Generate a unique message ID
            String messageId = UUID.randomUUID().toString();

            // Build success response
            SendEmailResponse response = SendEmailResponse.newBuilder()
                    .setSuccess(true)
                    .setMessageId(messageId)
                    .build();

            // Send response back to client
            responseObserver.onNext(response);
            responseObserver.onCompleted();

            logger.info("Email sent successfully. Message ID: {}", messageId);
        } catch (Exception e) {
            logger.error("Failed to send email", e);

            // Build error response
            SendEmailResponse response = SendEmailResponse.newBuilder()
                    .setSuccess(false)
                    .setMessageId("Failed to send email: " + e.getMessage())
                    .build();

            responseObserver.onError(
                io.grpc.Status.INTERNAL
                    .withDescription("Failed to send email: " + e.getMessage())
                    .asRuntimeException()
            );
        }

    }
}
