module Main exposing (..)

import Array exposing (Array, empty, toList)
import Html exposing (..)
import Html.Events exposing (..)
import Html.Attributes exposing (value)
import WebSocket
import Json.Encode exposing (Value, encode)
import Json.Decode exposing (decodeString, Decoder)
import Json.Decode.Pipeline exposing (decode, required)


serverAddr : String
serverAddr =
    "ws://localhost:12345/ws"


main : Program Never Model Msg
main =
    Html.program
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }



-- MODEL


type alias S2CMessage =
    { messages : Array Message }

type alias Message = 
    { user : String
    , body : String
    }

type alias C2SMessage =
    { user : String
    , body : String
    }


type alias Model =
    { inputUser : String
    , inputBody : String
    , messages : S2CMessage
    }


init : ( Model, Cmd Msg )
init =
    ( Model "" "" (S2CMessage empty), Cmd.none )


msgDecoder : Decoder Message
msgDecoder =
    decode Message
        |> required "user" Json.Decode.string
        |> required "body" Json.Decode.string

messageDecoder : Decoder S2CMessage
messageDecoder =
    decode S2CMessage
        |> required "messages" (Json.Decode.array msgDecoder)


messageEncoder : C2SMessage -> Value
messageEncoder msg =
    Json.Encode.object
        [ ( "user", Json.Encode.string msg.user )
        , ( "body", Json.Encode.string msg.body )
        ]



-- UPDATE


type Msg
    = InputUser String
    | InputBody String
    | Send
    | NewMessage String


update : Msg -> Model -> ( Model, Cmd Msg )
update msg { inputUser, inputBody, messages } =
    case msg of
        InputUser newUser ->
            ( Model newUser inputBody messages, Cmd.none )

        InputBody newBody ->
            ( Model inputUser newBody messages, Cmd.none )

        Send ->
            ( Model inputUser "" messages
            , WebSocket.send serverAddr
                (encode 0 (messageEncoder (C2SMessage inputUser inputBody)))
            )

        NewMessage str ->
            case decodeString messageDecoder str of
                Ok smsg ->
                    ( Model inputUser inputBody smsg, Cmd.none )

                Err err ->
                    ( Model inputUser inputBody messages, Debug.crash err )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    WebSocket.listen serverAddr NewMessage



-- VIEW


view : Model -> Html Msg
view model =
    div []
        [ input [ onInput InputUser, value model.inputUser ] []
        , input [ onInput InputBody, value model.inputBody ] []
        , button [ onClick Send ] [ text "Send" ]
        , div [] (toList (Array.map viewMessage model.messages.messages))
        ]


viewMessage : Message -> Html msg
viewMessage msg =
    div [] [ text (msg.user ++ " says " ++ msg.body ++ "."), br [] [] ]
