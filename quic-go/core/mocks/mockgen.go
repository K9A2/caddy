package mocks

//go:generate sh -c "mockgen -package mockquic -destination quic/stream.go github.com/lucas-clemente/quic-go Stream && goimports -w quic/stream.go"
//go:generate sh -c "mockgen -package mockquic -destination quic/session.go github.com/lucas-clemente/quic-go Session && goimports -w quic/session.go"
//go:generate sh -c "mockgen -package mockquic -destination quic/listener.go github.com/lucas-clemente/quic-go Listener && goimports -w quic/listener.go"
//go:generate sh -c "../mockgen_internal.sh mocks short_header_sealer.go github.com/lucas-clemente/quic-go/core/handshake ShortHeaderSealer"
//go:generate sh -c "../mockgen_internal.sh mocks short_header_opener.go github.com/lucas-clemente/quic-go/core/handshake ShortHeaderOpener"
//go:generate sh -c "../mockgen_internal.sh mocks long_header_opener.go github.com/lucas-clemente/quic-go/core/handshake LongHeaderOpener"
//go:generate sh -c "../mockgen_internal.sh mocks crypto_setup.go github.com/lucas-clemente/quic-go/core/handshake CryptoSetup"
//go:generate sh -c "../mockgen_internal.sh mocks stream_flow_controller.go github.com/lucas-clemente/quic-go/core/flowcontrol StreamFlowController"
//go:generate sh -c "../mockgen_internal.sh mockackhandler ackhandler/sent_packet_handler.go github.com/lucas-clemente/quic-go/core/ackhandler SentPacketHandler"
//go:generate sh -c "../mockgen_internal.sh mockackhandler ackhandler/received_packet_handler.go github.com/lucas-clemente/quic-go/core/ackhandler ReceivedPacketHandler"
//go:generate sh -c "../mockgen_internal.sh mocks congestion.go github.com/lucas-clemente/quic-go/core/congestion SendAlgorithmWithDebugInfos"
//go:generate sh -c "../mockgen_internal.sh mocks connection_flow_controller.go github.com/lucas-clemente/quic-go/core/flowcontrol ConnectionFlowController"
