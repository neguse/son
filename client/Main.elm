module Main exposing (..)

import Html exposing (..)
import Html.Events exposing (..)
import Html.Attributes exposing (value)
import WebSocket
import Json.Encode exposing (Value, encode)
import Json.Decode exposing (decodeString, Decoder)
import Json.Decode.Pipeline exposing (decode, required)


serverAddr : String
serverAddr =
    "ws://localhost:12345/echo"


main : Program Never Model Msg
main =
    Html.program
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }



-- MODEL


type alias Message =
    { user : String
    , body : String
    }


type alias Model =
    { inputUser : String
    , inputBody : String
    , messages : List Message
    }


init : ( Model, Cmd Msg )
init =
    ( Model "" "" [], Cmd.none )


messageDecoder : Decoder Message
messageDecoder =
    decode Message
        |> required "user" Json.Decode.string
        |> required "body" Json.Decode.string


messageEncoder : Message -> Value
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
                (encode 0 (messageEncoder (Message inputUser inputBody)))
            )

        NewMessage str ->
            case decodeString messageDecoder str of
                Ok msg ->
                    ( Model inputUser inputBody (msg :: messages), Cmd.none )

                Err err ->
                    ( Model inputUser inputBody ((Message "error" err) :: messages), Cmd.none )



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
        , div [] (List.map viewMessage model.messages)
        ]


viewMessage : Message -> Html msg
viewMessage msg =
    div [] [ text (msg.user ++ " says " ++ msg.body ++ "."), br [] [] ]
